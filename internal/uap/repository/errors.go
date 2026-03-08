package repository

import "errors"

var (
	ErrClaimRawBatch     = errors.New("uap.repository: failed to claim raw batch")
	ErrUpdateRawBatch    = errors.New("uap.repository: failed to update raw batch")
	ErrRawBatchNotFound  = errors.New("uap.repository: raw batch not found")
)
