package kafka

import (
	"fmt"
	"sync"

	"ingest-srv/config"

	"github.com/smap-hcmut/shared-libs/go/kafka"
)

var (
	consumerInstance kafka.IConsumer
	consumerOnce     sync.Once
	consumerMu       sync.RWMutex
	consumerInitErr  error
)

// ConnectConsumer initializes and connects to Kafka Consumer Group using singleton pattern.
// Safe for concurrent use. Returns existing instance if already initialized.
func ConnectConsumer(cfg config.KafkaConfig) (kafka.IConsumer, error) {
	consumerMu.Lock()
	defer consumerMu.Unlock()

	if consumerInstance != nil {
		return consumerInstance, nil
	}

	if consumerInitErr != nil {
		consumerOnce = sync.Once{}
		consumerInitErr = nil
	}

	var err error
	consumerOnce.Do(func() {
		if cfg.GroupID == "" {
			err = fmt.Errorf("GroupID is required for Kafka consumer")
			consumerInitErr = err
			return
		}

		clientCfg := kafka.ConsumerConfig{
			Brokers: cfg.Brokers,
			GroupID: cfg.GroupID,
		}

		client, e := kafka.NewConsumer(clientCfg)
		if e != nil {
			err = fmt.Errorf("failed to initialize Kafka consumer: %w", e)
			consumerInitErr = err
			return
		}

		consumerInstance = client
	})

	return consumerInstance, err
}

// GetConsumer returns the singleton Kafka consumer instance.
// Panics if consumer is not initialized. Call ConnectConsumer() first.
func GetConsumer() kafka.IConsumer {
	consumerMu.RLock()
	defer consumerMu.RUnlock()

	if consumerInstance == nil {
		panic("Kafka consumer not initialized. Call ConnectConsumer() first")
	}
	return consumerInstance
}

// DisconnectConsumer closes the Kafka consumer and resets the singleton.
func DisconnectConsumer() error {
	consumerMu.Lock()
	defer consumerMu.Unlock()

	if consumerInstance != nil {
		if err := consumerInstance.Close(); err != nil {
			return err
		}
		consumerInstance = nil
		consumerOnce = sync.Once{}
		consumerInitErr = nil
	}
	return nil
}
