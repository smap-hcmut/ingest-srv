package kafka

import (
	"fmt"
	"sync"

	"ingest-srv/config"

	"github.com/smap-hcmut/shared-libs/go/kafka"
)

var (
	producerInstance kafka.IProducer
	producerOnce     sync.Once
	producerMu       sync.RWMutex
	producerInitErr  error
)

// ConnectProducer initializes and connects to Kafka Producer using singleton pattern.
// Safe for concurrent use. Returns existing instance if already initialized.
func ConnectProducer(cfg config.KafkaConfig) (kafka.IProducer, error) {
	producerMu.Lock()
	defer producerMu.Unlock()

	if producerInstance != nil {
		return producerInstance, nil
	}

	if producerInitErr != nil {
		producerOnce = sync.Once{}
		producerInitErr = nil
	}

	var err error
	producerOnce.Do(func() {
		clientCfg := kafka.Config{
			Brokers: cfg.Brokers,
			Topic:   cfg.Topic,
		}

		client, e := kafka.NewProducer(clientCfg)
		if e != nil {
			err = fmt.Errorf("failed to initialize Kafka producer: %w", e)
			producerInitErr = err
			return
		}

		producerInstance = client
	})

	return producerInstance, err
}

// GetProducer returns the singleton Kafka producer instance.
// Panics if producer is not initialized. Call ConnectProducer() first.
func GetProducer() kafka.IProducer {
	producerMu.RLock()
	defer producerMu.RUnlock()

	if producerInstance == nil {
		panic("Kafka producer not initialized. Call ConnectProducer() first")
	}
	return producerInstance
}

// ProducerHealthCheck checks if Kafka producer is initialized and healthy.
func ProducerHealthCheck() error {
	producerMu.RLock()
	defer producerMu.RUnlock()

	if producerInstance == nil {
		return fmt.Errorf("Kafka producer not initialized")
	}
	return producerInstance.HealthCheck()
}

// DisconnectProducer closes the Kafka producer and resets the singleton.
func DisconnectProducer() error {
	producerMu.Lock()
	defer producerMu.Unlock()

	if producerInstance != nil {
		if err := producerInstance.Close(); err != nil {
			return err
		}
		producerInstance = nil
		producerOnce = sync.Once{}
		producerInitErr = nil
	}
	return nil
}
