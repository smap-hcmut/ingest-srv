package consumer

import (
	"context"
	"encoding/json"

	amqp "github.com/rabbitmq/amqp091-go"

	"smap-collector/internal/models"
	"smap-collector/internal/results"
)

func (c Consumer) resultWorker(d amqp.Delivery) {
	ctx := context.Background()
	c.l.Info(ctx, "results.delivery.rabbitmq.consumer.resultWorker")

	var res models.CrawlerResult
	if err := json.Unmarshal(d.Body, &res); err != nil {
		c.l.Warnf(ctx, "results.consumer.Unmarshal: %v", err)
		d.Ack(false)
		return
	}

	if err := c.uc.HandleResult(ctx, res); err != nil {
		if err == results.ErrInvalidInput {
			c.l.Warnf(ctx, "results.consumer.HandleResult invalid: %v", err)
			d.Ack(false)
			return
		}
		c.l.Errorf(ctx, "results.consumer.HandleResult: %v", err)
		// For temporary errors, we might want to nack and requeue
		// For now, we'll ack to prevent infinite loops
		d.Ack(false)
		return
	}

	c.l.Infof(ctx, "results.consumer.HandleResult completed successfully")
	d.Ack(false)
}
