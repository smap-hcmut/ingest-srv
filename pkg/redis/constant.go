package redis

import "time"

const (
	// DefaultConnectTimeout is the timeout for initial connection ping.
	DefaultConnectTimeout = 5 * time.Second
)
