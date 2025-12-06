package dispatcher

import "smap-collector/internal/models"

// Options for dispatcher defaults.
type Options struct {
	DefaultMaxAttempts int
	SchemaVersion      int
	PlatformQueues     map[models.Platform]string
}
