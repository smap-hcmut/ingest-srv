package kafka

import (
	"github.com/IBM/sarama"
)

// Config holds configuration for Kafka producer.
type Config struct {
	Brokers []string
	Topic   string
}

// producerImpl implements IProducer.
type producerImpl struct {
	producer sarama.SyncProducer
	topic    string
}

// ConsumerConfig holds configuration for Kafka consumer group.
type ConsumerConfig struct {
	Brokers []string
	GroupID string
}

// consumerImpl implements IConsumer.
type consumerImpl struct {
	group sarama.ConsumerGroup
}
