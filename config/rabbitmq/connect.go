package rabbitmq

import (
	"fmt"
	"sync"

	"ingest-srv/config"
	"ingest-srv/pkg/rabbitmq"
)

var (
	instance rabbitmq.IRabbitMQ
	once     sync.Once
	mu       sync.RWMutex
	initErr  error
)

// Connect initializes and connects to RabbitMQ using singleton pattern.
func Connect(cfg config.RabbitMQConfig) (rabbitmq.IRabbitMQ, error) {
	mu.Lock()
	defer mu.Unlock()

	if instance != nil {
		return instance, nil
	}

	if initErr != nil {
		once = sync.Once{}
		initErr = nil
	}

	var err error
	once.Do(func() {
		conn, e := rabbitmq.NewRabbitMQ(cfg.URL, cfg.RetryWithoutTimeout)
		if e != nil {
			err = fmt.Errorf("failed to initialize RabbitMQ connection: %w", e)
			initErr = err
			return
		}
		instance = conn
	})

	return instance, err
}

// GetClient returns the singleton RabbitMQ instance.
func GetClient() rabbitmq.IRabbitMQ {
	mu.RLock()
	defer mu.RUnlock()

	if instance == nil {
		panic("RabbitMQ client not initialized. Call Connect() first")
	}
	return instance
}

// HealthCheck checks if RabbitMQ connection is ready.
func HealthCheck() error {
	mu.RLock()
	defer mu.RUnlock()

	if instance == nil {
		return fmt.Errorf("RabbitMQ client not initialized")
	}
	if !instance.IsReady() {
		return fmt.Errorf("RabbitMQ connection is not ready")
	}
	return nil
}

// Disconnect closes RabbitMQ connection and resets singleton.
func Disconnect() {
	mu.Lock()
	defer mu.Unlock()

	if instance != nil {
		instance.Close()
		instance = nil
		once = sync.Once{}
		initErr = nil
	}
}
