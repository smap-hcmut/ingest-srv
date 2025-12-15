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

// ProgressRequest là request body cho progress webhook.
// Gửi tới POST /internal/progress/callback
// Two-phase format: crawl + analyze progress
type ProgressRequest struct {
	ProjectID              string        `json:"project_id"`
	UserID                 string        `json:"user_id"`
	Status                 string        `json:"status"` // INITIALIZING, PROCESSING, DONE, FAILED
	Crawl                  PhaseProgress `json:"crawl"`
	Analyze                PhaseProgress `json:"analyze"`
	OverallProgressPercent float64       `json:"overall_progress_percent"`
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
