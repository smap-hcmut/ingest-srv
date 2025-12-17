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

// Redis hash field names - Hybrid state structure
const (
	FieldStatus = "status"

	// Task-level fields (for completion check)
	// Mỗi response từ Crawler = 1 task
	FieldTasksTotal  = "tasks_total"
	FieldTasksDone   = "tasks_done"
	FieldTasksErrors = "tasks_errors"

	// Item-level fields (for progress display)
	// Số items thực tế crawl được từ platform
	FieldItemsExpected = "items_expected"
	FieldItemsActual   = "items_actual"
	FieldItemsErrors   = "items_errors"

	// Analyze phase fields (unchanged)
	FieldAnalyzeTotal  = "analyze_total"
	FieldAnalyzeDone   = "analyze_done"
	FieldAnalyzeErrors = "analyze_errors"

	// Legacy crawl phase fields (for backward compatibility)
	// Deprecated: Use task-level and item-level fields instead
	FieldCrawlTotal  = "crawl_total"
	FieldCrawlDone   = "crawl_done"
	FieldCrawlErrors = "crawl_errors"
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
