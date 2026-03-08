package uap

import "context"

// UseCase defines UAP parsing operations.
type UseCase interface {
	ParseAndStoreRawBatch(ctx context.Context, input ParseAndStoreRawBatchInput) error
}
