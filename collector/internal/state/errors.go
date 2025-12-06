package state

import "errors"

var (
	ErrStateNotFound       = errors.New("project state not found")
	ErrInvalidProjectID    = errors.New("invalid project ID")
	ErrInvalidStatus       = errors.New("invalid project status")
	ErrInvalidTotal        = errors.New("invalid total value")
	ErrRedisConnection     = errors.New("redis connection error")
	ErrRedisOperation      = errors.New("redis operation error")
	ErrUserMappingNotFound = errors.New("user mapping not found")
	ErrStateAlreadyExists  = errors.New("project state already exists")
)
