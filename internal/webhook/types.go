package webhook

import (
	"errors"
)

// PhaseProgress chứa progress của một phase (crawl hoặc analyze).
type PhaseProgress struct {
	Total           int64   `json:"total"`
	Done            int64   `json:"done"`
	Errors          int64   `json:"errors"`
	ProgressPercent float64 `json:"progress_percent"`
}

// TaskProgress chứa progress dựa trên task-level (for completion tracking).
// Mỗi response từ Crawler = 1 task.
type TaskProgress struct {
	Total   int64   `json:"total"`   // Số tasks dispatch (keywords × platforms)
	Done    int64   `json:"done"`    // Số tasks hoàn thành
	Errors  int64   `json:"errors"`  // Số tasks failed
	Percent float64 `json:"percent"` // (done + errors) / total * 100
}

// ItemProgress chứa progress dựa trên item-level (for progress display).
// Số items thực tế crawl được từ platform.
type ItemProgress struct {
	Expected int64   `json:"expected"` // tasks × limit_per_keyword
	Actual   int64   `json:"actual"`   // Số items crawl thành công
	Errors   int64   `json:"errors"`   // Số items crawl thất bại
	Percent  float64 `json:"percent"`  // (actual + errors) / expected * 100
}

// ProgressRequest là request body cho progress webhook.
// Gửi tới POST /internal/progress/callback
// Hybrid format: task-level + item-level + analyze progress
type ProgressRequest struct {
	ProjectID string `json:"project_id"`
	UserID    string `json:"user_id"`
	Status    string `json:"status"` // INITIALIZING, PROCESSING, DONE, FAILED

	// Task-level progress (for completion tracking)
	Tasks TaskProgress `json:"tasks"`

	// Item-level progress (for progress display)
	Items ItemProgress `json:"items"`

	// Legacy crawl progress (for backward compatibility)
	Crawl PhaseProgress `json:"crawl"`

	// Analyze phase progress
	Analyze PhaseProgress `json:"analyze"`

	OverallProgressPercent float64 `json:"overall_progress_percent"`
}

// IsValid kiểm tra request có hợp lệ không.
func (r *ProgressRequest) IsValid() bool {
	return r.ProjectID != "" && r.UserID != "" && r.Status != ""
}

// Webhook errors
var (
	// ErrInvalidRequest khi request không hợp lệ.
	ErrInvalidRequest = errors.New("invalid webhook request")
)
