package consumer

import (
	"context"
	"encoding/json"

	amqp "github.com/rabbitmq/amqp091-go"

	"smap-collector/internal/dispatcher"
	"smap-collector/internal/dispatcher/delivery/rabbitmq"
	"smap-collector/internal/models"
)

// ConsumeProjectEvents starts consuming project.created events from smap.events exchange.
// This is the new event-driven architecture consumer.
func (c Consumer) ConsumeProjectEvents() {
	go c.consume(
		rabbitmq.SMAPEventsExchangeArgs,
		rabbitmq.QueueProjectCreated,
		rabbitmq.RoutingKeyProjectCreated,
		c.projectEventWorker,
	)
}

// projectEventWorker handles ProjectCreatedEvent messages.
func (c Consumer) projectEventWorker(d amqp.Delivery) {
	ctx := context.Background()
	c.l.Info(ctx, "dispatcher.delivery.rabbitmq.consumer.projectEventWorker")

	// Try to parse as ProjectCreatedEvent first
	var event models.ProjectCreatedEvent
	if err := json.Unmarshal(d.Body, &event); err != nil {
		c.l.Warnf(ctx, "dispatcher.delivery.rabbitmq.consumer.projectEventWorker: failed to unmarshal ProjectCreatedEvent: %v", err)
		d.Ack(false)
		return
	}

	// Validate event
	if !event.IsValid() {
		c.l.Warnf(ctx, "dispatcher.delivery.rabbitmq.consumer.projectEventWorker: invalid event - missing required fields: event_id=%s, project_id=%s, user_id=%s",
			event.EventID, event.Payload.ProjectID, event.Payload.UserID)
		d.Ack(false)
		return
	}

	c.l.Infof(ctx, "dispatcher.delivery.rabbitmq.consumer.projectEventWorker: received event - event_id=%s, project_id=%s, user_id=%s, brand=%s, keywords=%d",
		event.EventID,
		event.Payload.ProjectID,
		event.Payload.UserID,
		event.Payload.BrandName,
		len(event.Payload.BrandKeywords)+countCompetitorKeywords(event.Payload.CompetitorKeywordsMap),
	)

	// Handle the event using usecase (includes state updates and webhook notifications)
	if err := c.uc.HandleProjectCreatedEvent(ctx, event); err != nil {
		if err == dispatcher.ErrInvalidInput || err == dispatcher.ErrUnknownRoute {
			c.l.Warnf(ctx, "dispatcher.delivery.rabbitmq.consumer.projectEventWorker: invalid event: %v", err)
			d.Ack(false)
			return
		}
		c.l.Errorf(ctx, "dispatcher.delivery.rabbitmq.consumer.projectEventWorker: failed to handle event: %v", err)
		// Nack with requeue for temporary errors
		d.Nack(false, true)
		return
	}

	c.l.Infof(ctx, "dispatcher.delivery.rabbitmq.consumer.projectEventWorker: successfully handled event - event_id=%s, project_id=%s",
		event.EventID, event.Payload.ProjectID)
	d.Ack(false)
}

// countCompetitorKeywords counts total keywords from competitor map.
func countCompetitorKeywords(m map[string][]string) int {
	count := 0
	for _, keywords := range m {
		count += len(keywords)
	}
	return count
}
