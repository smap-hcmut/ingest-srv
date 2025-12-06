package usecase

import "errors"

var (
	// ErrInvalidProjectEvent khi event không hợp lệ.
	ErrInvalidProjectEvent = errors.New("invalid project created event")

	// ErrPublishFailed khi publish task thất bại.
	ErrPublishFailed = errors.New("publish task failed")
)
