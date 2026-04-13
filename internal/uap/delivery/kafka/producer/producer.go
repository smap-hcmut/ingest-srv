package producer

import (
	"context"
	"strings"

	"ingest-srv/internal/uap"
	uapKafka "ingest-srv/internal/uap/delivery/kafka"

	"github.com/smap-hcmut/shared-libs/go/constants"
)

// Publish publishes a parsed UAP record to Kafka.
func (p *publisher) Publish(ctx context.Context, input uap.PublishUAPInput) error {
	if p.producer == nil {
		p.logger.Warn(ctx, "uap.delivery.kafka.producer.Publish: producer is nil")
		return nil
	}

	body, err := uapKafka.MarshalUAPRecord(input.Record)
	if err != nil {
		return err
	}

	key := []byte(strings.TrimSpace(input.Record.Identity.UAPID))
	return p.producer.Publish(key, body)
}

func (p *publisher) Topic() string {
	return constants.TopicCollectorOutput
}

// Close closes the underlying Kafka producer if needed.
func (p *publisher) Close() error {
	if p.producer == nil {
		return nil
	}

	return p.producer.Close()
}
