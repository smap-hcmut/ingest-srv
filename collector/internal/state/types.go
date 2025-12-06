package state

import "time"

// Redis key schema constants
const (
	// KeyPrefix là prefix cho tất cả state keys
	KeyPrefix = "smap:proj:"

	// UserMappingPrefix là prefix cho user mapping keys
	UserMappingPrefix = "smap:user:"

	// DefaultTTL là TTL mặc định cho state keys (7 ngày)
	DefaultTTL = 7 * 24 * time.Hour
)

// Redis hash field names
const (
	FieldStatus = "status"
	FieldTotal  = "total"
	FieldDone   = "done"
	FieldErrors = "errors"
)

// BuildStateKey tạo Redis key cho project state.
// Format: smap:proj:{projectID}
func BuildStateKey(projectID string) string {
	return KeyPrefix + projectID
}

// BuildUserMappingKey tạo Redis key cho user mapping.
// Format: smap:user:{projectID}
func BuildUserMappingKey(projectID string) string {
	return UserMappingPrefix + projectID
}

// Options chứa các tùy chọn cho state usecase.
type Options struct {
	// TTL cho state keys, default 7 ngày
	TTL time.Duration

	// RedisDB là database number cho state (default: 1)
	RedisDB int
}

// DefaultOptions trả về options mặc định.
func DefaultOptions() Options {
	return Options{
		TTL:     DefaultTTL,
		RedisDB: 1,
	}
}
