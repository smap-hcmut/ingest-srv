package redis

import "errors"

var (
	ErrHostRequired = errors.New("redis: host is required")
	ErrInvalidPort  = errors.New("redis: invalid port")
)
