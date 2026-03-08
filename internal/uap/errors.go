package uap

import "errors"

var (
	ErrInvalidRawBatchInput = errors.New("uap: invalid raw batch input")
	ErrParseRawPayload      = errors.New("uap: failed to parse raw payload")
)
