package usecase

import "errors"

var (
	// ErrUpdateCompletionFailed khi update completion status thất bại.
	ErrUpdateCompletionFailed = errors.New("failed to update completion status")

	// ErrInvalidUserID khi user ID không hợp lệ.
	ErrInvalidUserID = errors.New("invalid user ID")
)
