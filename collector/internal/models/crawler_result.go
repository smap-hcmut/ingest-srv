package models

// CrawlerStatus mô tả trạng thái trả về của worker.
type CrawlerStatus string

const (
	CrawlerStatusSuccess CrawlerStatus = "success"
	CrawlerStatusSkipped CrawlerStatus = "skipped"
	CrawlerStatusFailed  CrawlerStatus = "failed"
)

// CrawlerResult là payload worker gửi ngược collector.
// Format mới chỉ có 2 fields: success và payload (array of content items).
// Metadata như job_id, platform được lấy từ payload[].meta
type CrawlerResult struct {
	Success bool `json:"success"`
	Payload any  `json:"payload"` // Array of content items with meta, content, interaction, author, comments
}

// ResultMetrics thống kê kết quả crawl.
type ResultMetrics struct {
	Documents  int64 `json:"documents,omitempty"`
	Bytes      int64 `json:"bytes,omitempty"`
	DurationMs int64 `json:"duration_ms,omitempty"`
}

// ResultError chứa mã lỗi máy đọc được từ worker.
type ResultError struct {
	Code    string `json:"code"`
	Message string `json:"message,omitempty"`
}

// ============================================================================
// Enhanced Crawler Response (for new Crawler format with limit_info and stats)
// ============================================================================

// EnhancedCrawlerResult là response format mới từ Crawler với limit_info và stats.
// Collector sẽ parse format này trước, fallback về CrawlerResult nếu không có.
type EnhancedCrawlerResult struct {
	Success   bool        `json:"success"`
	TaskType  string      `json:"task_type,omitempty"`
	LimitInfo *LimitInfo  `json:"limit_info,omitempty"`
	Stats     *CrawlStats `json:"stats,omitempty"`
	Error     *CrawlError `json:"error,omitempty"`
	Payload   any         `json:"payload"`
}

// LimitInfo chứa thông tin về limits và actual results từ Crawler.
// Giúp Collector biết platform có bị giới hạn không.
type LimitInfo struct {
	RequestedLimit  int  `json:"requested_limit"`  // Limit được request từ Collector
	AppliedLimit    int  `json:"applied_limit"`    // Limit thực tế Crawler áp dụng
	TotalFound      int  `json:"total_found"`      // Số items tìm được trên platform
	PlatformLimited bool `json:"platform_limited"` // Platform có bị giới hạn không
}

// CrawlStats chứa statistics về crawl results.
// Dùng để update item-level state trong Redis.
type CrawlStats struct {
	Successful     int     `json:"successful"`      // Số items crawl thành công
	Failed         int     `json:"failed"`          // Số items crawl thất bại
	Skipped        int     `json:"skipped"`         // Số items bị skip
	CompletionRate float64 `json:"completion_rate"` // Tỷ lệ hoàn thành (0.0 - 1.0)
}

// CrawlError chứa error details khi crawl fail.
type CrawlError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// IsRetryable kiểm tra error có thể retry không.
// Một số lỗi như AUTH_FAILED, INVALID_KEYWORD không nên retry.
func (e *CrawlError) IsRetryable() bool {
	if e == nil {
		return false
	}
	switch e.Code {
	case "AUTH_FAILED", "INVALID_KEYWORD", "BLOCKED", "RATE_LIMITED_PERMANENT":
		return false
	default:
		return true
	}
}
