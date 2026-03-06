package repository

import "errors"

// Repository-level errors.
var (
	ErrFailedToInsert = errors.New("datasource.repository: failed to insert")
	ErrFailedToGet    = errors.New("datasource.repository: failed to get")
	ErrFailedToUpdate = errors.New("datasource.repository: failed to update")
	ErrFailedToDelete = errors.New("datasource.repository: failed to delete")
	ErrFailedToList   = errors.New("datasource.repository: failed to list")

	// CrawlTarget errors.
	ErrTargetFailedToInsert = errors.New("datasource.repository: failed to insert target")
	ErrTargetNotFound       = errors.New("datasource.repository: target not found")
	ErrTargetFailedToUpdate = errors.New("datasource.repository: failed to update target")
	ErrTargetFailedToDelete = errors.New("datasource.repository: failed to delete target")
	ErrTargetFailedToList   = errors.New("datasource.repository: failed to list targets")
)
