package consumer

import (
	"context"
	"encoding/json"

	amqp "github.com/rabbitmq/amqp091-go"

	"smap-collector/internal/dispatcher"
	"smap-collector/internal/models"
)

func (c Consumer) dispatchWorker(d amqp.Delivery) {
	ctx := context.Background()
	c.l.Info(ctx, "dispatcher.delivery.rabbitmq.consumer.dispatchWorker")

	var req models.CrawlRequest
	if err := json.Unmarshal(d.Body, &req); err != nil {
		c.l.Warnf(ctx, "dispatcher.consumer.Unmarshal: %v", err)
		d.Ack(false)
		return
	}

	tasks, err := c.uc.Dispatch(ctx, req)
	if err != nil {
		if err == dispatcher.ErrInvalidInput || err == dispatcher.ErrUnknownRoute {
			c.l.Warnf(ctx, "dispatcher.consumer.Dispatch invalid: %v", err)
			d.Ack(false)
			return
		}
		c.l.Errorf(ctx, "dispatcher.consumer.Dispatch: %v", err)
		d.Ack(false)
		return
	}

	c.l.Infof(ctx, "dispatcher.consumer.Dispatch published %d task(s)", len(tasks))
	d.Ack(false)
}
