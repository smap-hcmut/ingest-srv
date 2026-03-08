package uap

import "context"

// UseCase defines UAP parsing operations.
type UseCase interface {
	ParseAndStoreRawBatch(ctx context.Context, input ParseAndStoreRawBatchInput) error
}

// Publisher publishes parsed UAP messages downstream.
type Publisher interface {
	Publish(ctx context.Context, input PublishUAPInput) error
	Close() error
}
