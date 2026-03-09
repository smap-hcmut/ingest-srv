package repository

import "errors"

// Repository errors for dryrun persistence.
var (
	ErrFailedToInsert = errors.New("dryrun.repository: failed to insert")
	ErrFailedToUpdate = errors.New("dryrun.repository: failed to update")
	ErrFailedToGet    = errors.New("dryrun.repository: failed to get")
	ErrFailedToList   = errors.New("dryrun.repository: failed to list")
	ErrNotFound       = errors.New("dryrun.repository: result not found")
)
