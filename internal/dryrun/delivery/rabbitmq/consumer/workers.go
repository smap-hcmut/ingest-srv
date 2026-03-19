package consumer

import (
	"context"
	"encoding/json"
	"errors"

	"ingest-srv/internal/dryrun"
	dryrunRabbit "ingest-srv/internal/dryrun/delivery/rabbitmq"

	amqp "github.com/rabbitmq/amqp091-go"
)

func (c Consumer) handleCompletionWorker(delivery amqp.Delivery) {
	ctx := context.Background()

	var message dryrunRabbit.CompletionMessage
	if err := json.Unmarshal(delivery.Body, &message); err != nil {
		c.l.Warnf(ctx, "dryrun.delivery.rabbitmq.consumer.handleCompletionWorker.invalid_json: %v", err)
		_ = delivery.Ack(false)
		return
	}

	if err := c.dryrunUC.HandleCompletion(ctx, message.ToHandleCompletionInput()); err != nil {
		if errors.Is(err, dryrun.ErrCompletionTaskNotFound) || errors.Is(err, dryrun.ErrInvalidCompletionInput) {
			c.l.Warnf(ctx, "dryrun.delivery.rabbitmq.consumer.handleCompletionWorker.invalid_or_unknown: task_id=%s err=%v", message.TaskID, err)
			_ = delivery.Ack(false)
			return
		}

		c.l.Errorf(ctx, "dryrun.delivery.rabbitmq.consumer.handleCompletionWorker.HandleCompletion: task_id=%s err=%v", message.TaskID, err)
		_ = delivery.Nack(false, true)
		return
	}

	c.l.Infof(ctx, "dryrun.delivery.rabbitmq.consumer.handleCompletionWorker.success: task_id=%s", message.TaskID)
	_ = delivery.Ack(false)
}
