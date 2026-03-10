package kafka

import (
	"context"

	"github.com/IBM/sarama"
)

// IProducer defines the interface for Kafka producer.
// Implementations are safe for concurrent use.
type IProducer interface {
	Publish(key, value []byte) error
	Close() error
	HealthCheck() error
}

// IConsumer defines the interface for Kafka consumer group.
// Wraps sarama.ConsumerGroup for easier testing and management.
type IConsumer interface {
	// Consume starts consuming from topics. Uses background context.
	Consume(topics []string, handler sarama.ConsumerGroupHandler) error
	// ConsumeWithContext starts consuming from topics with context. Blocks until context is cancelled.
	ConsumeWithContext(ctx context.Context, topics []string, handler sarama.ConsumerGroupHandler) error
	// Close closes the consumer group
	Close() error
	// Errors returns a channel of errors from the consumer
	Errors() <-chan error
}

// NewProducer creates a new Kafka producer. Returns the interface.
func NewProducer(cfg Config) (IProducer, error) {
	if err := validateProducerConfig(cfg); err != nil {
		return nil, err
	}
	return newProducerImpl(cfg)
}

// NewConsumer creates a new Kafka consumer group. Returns the interface.
func NewConsumer(cfg ConsumerConfig) (IConsumer, error) {
	if err := validateConsumerConfig(cfg); err != nil {
		return nil, err
	}
	return newConsumerImpl(cfg)
}
