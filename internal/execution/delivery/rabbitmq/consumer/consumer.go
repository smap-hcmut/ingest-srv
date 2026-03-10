package consumer

import (
	"context"
	"fmt"
	"log"

	executionRabbit "ingest-srv/internal/execution/delivery/rabbitmq"
	"ingest-srv/pkg/rabbitmq"

	amqp "github.com/rabbitmq/amqp091-go"
)

func (c Consumer) Consume(ctx context.Context) error {
	return c.consume(ctx, executionRabbit.IngestTaskCompletionsQueue, executionRabbit.IngestTaskCompletionsConsumerName, c.handleCompletionWorker)
}

func catchPanic() {
	if r := recover(); r != nil {
		log.Printf("Recovered from panic in goroutine: %v", r)
	}
}

type workerFunc func(msg amqp.Delivery)

func (c Consumer) consume(ctx context.Context, queue rabbitmq.QueueArgs, consumerName string, worker workerFunc) error {
	defer catchPanic()

	if c.conn == nil {
		return fmt.Errorf("rabbitmq client is required")
	}
	if c.uc == nil {
		return fmt.Errorf("execution usecase is required")
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
				return nil
			}
			worker(msg)
		}
	}
}
