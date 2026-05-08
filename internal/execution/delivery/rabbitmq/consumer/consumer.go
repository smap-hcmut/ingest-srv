package consumer

import (
	"context"
	"errors"
	"fmt"
	"time"

	executionRabbit "ingest-srv/internal/execution/delivery/rabbitmq"

	"github.com/smap-hcmut/shared-libs/go/rabbitmq"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	consumerReconnectInitialBackoff = 1 * time.Second
	consumerReconnectMaxBackoff     = 30 * time.Second
)

var errConsumerDeliveryChannelClosed = errors.New("rabbitmq delivery channel closed")

func (c Consumer) Consume(ctx context.Context) error {
	return c.consume(ctx, executionRabbit.IngestTaskCompletionsQueue, executionRabbit.IngestTaskCompletionsConsumerName, c.handleCompletionWorker)
}

func catchPanicAsError(err *error) {
	if r := recover(); r != nil {
		*err = fmt.Errorf("panic in rabbitmq consumer: %v", r)
	}
}

type workerFunc func(msg amqp.Delivery)

func (c Consumer) consume(ctx context.Context, queue rabbitmq.QueueArgs, consumerName string, worker workerFunc) error {
	if c.conn == nil {
		return fmt.Errorf("rabbitmq client is required")
	}
	if c.execUC == nil {
		return fmt.Errorf("execution consumer usecase is required")
	}

	backoff := consumerReconnectInitialBackoff
	for {
		if ctx.Err() != nil {
			return nil
		}

		err := c.consumeOnce(ctx, queue, consumerName, worker)
		if ctx.Err() != nil {
			return nil
		}
		if err != nil {
			c.l.Warnf(context.Background(), "execution.rabbitmq.consumer.reconnect: queue=%s consumer=%s err=%v retry_in=%s", queue.Name, consumerName, err, backoff)
		} else {
			c.l.Warnf(context.Background(), "execution.rabbitmq.consumer.reconnect: queue=%s consumer=%s stopped unexpectedly retry_in=%s", queue.Name, consumerName, backoff)
		}

		select {
		case <-ctx.Done():
			return nil
		case <-time.After(backoff):
		}

		backoff *= 2
		if backoff > consumerReconnectMaxBackoff {
			backoff = consumerReconnectMaxBackoff
		}
	}
}

func (c Consumer) consumeOnce(ctx context.Context, queue rabbitmq.QueueArgs, consumerName string, worker workerFunc) (err error) {
	defer catchPanicAsError(&err)

	if !c.conn.IsReady() {
		return fmt.Errorf("rabbitmq connection is not ready")
	}

	ch, err := c.conn.Channel()
	if err != nil {
		return err
	}
	defer func() {
		_ = ch.Close()
	}()

	q, err := ch.QueueDeclare(queue)
	if err != nil {
		return err
	}

	msgs, err := ch.Consume(rabbitmq.ConsumeArgs{
		Queue:    q.Name,
		Consumer: consumerName,
		AutoAck:  false,
	})
	if err != nil {
		return err
	}

	c.l.Infof(ctx, "Queue %s is being consumed", q.Name)

	for {
		select {
		case <-ctx.Done():
			c.l.Infof(context.Background(), "Stopping consumer for queue %s", q.Name)
			return nil
		case msg, ok := <-msgs:
			if !ok {
				c.l.Warnf(context.Background(), "Consumer channel closed for queue %s", q.Name)
				return errConsumerDeliveryChannelClosed
			}
			if err := callWorkerSafely(worker, msg); err != nil {
				c.l.Errorf(context.Background(), "execution.rabbitmq.consumer.worker_panic: queue=%s err=%v", q.Name, err)
				_ = msg.Nack(false, true)
				continue
			}
		}
	}
}

func callWorkerSafely(worker workerFunc, msg amqp.Delivery) (err error) {
	defer catchPanicAsError(&err)
	worker(msg)
	return nil
}
