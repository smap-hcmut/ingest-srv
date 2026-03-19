package consumer

import (
	"context"
	"encoding/json"
	"errors"

	"ingest-srv/internal/execution"
	executionRabbit "ingest-srv/internal/execution/delivery/rabbitmq"

	amqp "github.com/rabbitmq/amqp091-go"
)

func (c Consumer) handleCompletionWorker(delivery amqp.Delivery) {
	ctx := context.Background()

	var message executionRabbit.CompletionMessage
	if err := json.Unmarshal(delivery.Body, &message); err != nil {
		c.l.Warnf(ctx, "execution.delivery.rabbitmq.consumer.handleCompletionWorker.invalid_json: %v", err)
		_ = delivery.Ack(false)
		return
	}

	if err := c.execUC.HandleCompletion(ctx, message.ToHandleCompletionInput()); err != nil {
		if errors.Is(err, execution.ErrInvalidCompletionInput) {
			c.l.Warnf(ctx, "execution.delivery.rabbitmq.consumer.handleCompletionWorker.invalid_input: task_id=%s err=%v", message.TaskID, err)
			_ = delivery.Ack(false)
			return
		}

		c.l.Errorf(ctx, "execution.delivery.rabbitmq.consumer.handleCompletionWorker.HandleCompletion: task_id=%s err=%v", message.TaskID, err)
		_ = delivery.Nack(false, true)
		return
	}

	c.l.Infof(ctx, "execution.delivery.rabbitmq.consumer.handleCompletionWorker.success: task_id=%s", message.TaskID)
	_ = delivery.Ack(false)
}
