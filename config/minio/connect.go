package minio

import (
	"context"
	"fmt"
	"sync"

	"ingest-srv/config"
	"ingest-srv/pkg/minio"
)

var (
	instance minio.MinIO
	once     sync.Once
	mu       sync.RWMutex
	initErr  error
)

// Connect initializes and connects to MinIO using singleton pattern.
func Connect(ctx context.Context, cfg *config.MinIOConfig) (minio.MinIO, error) {
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
		client, e := minio.NewMinIO(cfg)
		if e != nil {
			err = fmt.Errorf("failed to create MinIO client: %w", e)
			initErr = err
			return
		}
		if e := client.Connect(ctx); e != nil {
			err = fmt.Errorf("failed to connect to MinIO: %w", e)
			initErr = err
			return
		}
		instance = client
	})

	return instance, err
}

// GetClient returns the singleton MinIO client instance.
func GetClient() minio.MinIO {
	mu.RLock()
	defer mu.RUnlock()

	if instance == nil {
		panic("MinIO client not initialized. Call Connect() first")
	}
	return instance
}

// HealthCheck checks if MinIO connection is healthy
func HealthCheck(ctx context.Context) error {
	mu.RLock()
	defer mu.RUnlock()

	if instance == nil {
		return fmt.Errorf("MinIO client not initialized")
	}

	return instance.HealthCheck(ctx)
}

// Disconnect closes the MinIO client and resets the singleton.
func Disconnect() error {
	mu.Lock()
	defer mu.Unlock()

	if instance != nil {
		if err := instance.Close(); err != nil {
			return err
		}
		instance = nil
		once = sync.Once{}
		initErr = nil
	}
	return nil
}
