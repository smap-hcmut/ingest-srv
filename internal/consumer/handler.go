package consumer

import (
	"context"

	"ingest-srv/pkg/log"

	"github.com/IBM/sarama"
)

// NoopHandler consumes messages and only logs metadata.
type NoopHandler struct {
	l log.Logger
}

// NewNoopHandler returns a new no-op consumer handler.
func NewNoopHandler(l log.Logger) *NoopHandler {
	return &NoopHandler{l: l}
}

// Setup is run at the beginning of a new session.
func (h *NoopHandler) Setup(sarama.ConsumerGroupSession) error {
	h.l.Info(context.Background(), "Noop Kafka consumer setup completed")
	return nil
}

// Cleanup is run at the end of a session.
func (h *NoopHandler) Cleanup(sarama.ConsumerGroupSession) error {
	h.l.Info(context.Background(), "Noop Kafka consumer cleanup completed")
	return nil
}

// ConsumeClaim reads messages and marks them consumed.
func (h *NoopHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for msg := range claim.Messages() {
		h.l.Infof(
			context.Background(),
			"Noop consumer received message topic=%s partition=%d offset=%d key=%s",
			msg.Topic,
			msg.Partition,
			msg.Offset,
			string(msg.Key),
		)
		session.MarkMessage(msg, "")
	}
	return nil
}
