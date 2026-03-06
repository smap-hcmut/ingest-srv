package kafka

import (
	"time"

	"github.com/IBM/sarama"
)

const (
	// ProducerTimeout is the Kafka producer request timeout.
	ProducerTimeout = 10 * time.Second
	// ProducerRetryMax is the max producer retries.
	ProducerRetryMax = 3
)

var (
	// KafkaVersion is the sarama version used.
	KafkaVersion = sarama.V2_6_0_0
)
