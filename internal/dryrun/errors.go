package dryrun

import "errors"

// Domain errors for dryrun control-plane operations.
var (
	ErrSourceNotFound      = errors.New("dryrun source not found")
	ErrTargetNotFound      = errors.New("dryrun target not found")
	ErrTargetRequired      = errors.New("target_id is required for crawl dryrun")
	ErrTargetForbidden     = errors.New("target_id is not allowed for passive dryrun")
	ErrDryrunNotAllowed    = errors.New("dryrun is not allowed in current source state")
	ErrInvalidSampleLimit  = errors.New("sample_limit must be greater than 0")
	ErrResultNotFound      = errors.New("dryrun result not found")
	ErrCreateFailed        = errors.New("failed to create dryrun result")
	ErrGetFailed           = errors.New("failed to get dryrun result")
	ErrUpdateFailed        = errors.New("failed to update dryrun result")
	ErrListFailed          = errors.New("failed to list dryrun history")
)
