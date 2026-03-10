package producer

import (
	"context"

	"ingest-srv/internal/uap"
)

// Publish publishes a parsed UAP record to Kafka.
func (p *publisher) Publish(ctx context.Context, input uap.PublishUAPInput) error {
	if p.producer == nil {
		p.logger.Warn(ctx, "uap.delivery.kafka.producer.Publish: producer is nil")
		return nil
	}

	return p.producer.Publish(input.Key, input.Value)
}

// Close closes the underlying Kafka producer if needed.
func (p *publisher) Close() error {
	if p.producer == nil {
		return nil
	}

	return p.producer.Close()
}
