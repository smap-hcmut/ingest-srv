package webhook

import (
	"errors"
)

// ProgressRequest là request body cho progress webhook.
// Gửi tới POST /internal/progress/callback
type ProgressRequest struct {
	ProjectID string `json:"project_id"`
	UserID    string `json:"user_id"`
	Status    string `json:"status"` // INITIALIZING, CRAWLING, PROCESSING, DONE, FAILED
	Total     int64  `json:"total"`
	Done      int64  `json:"done"`
	Errors    int64  `json:"errors"`
}

// IsValid kiểm tra request có hợp lệ không.
func (r *ProgressRequest) IsValid() bool {
	return r.ProjectID != "" && r.UserID != "" && r.Status != ""
}

// ProgressPercent tính phần trăm tiến độ.
func (r *ProgressRequest) ProgressPercent() float64 {
	if r.Total <= 0 {
		return 0
	}
	return float64(r.Done) / float64(r.Total) * 100
}

// Webhook errors
var (
	// ErrInvalidRequest khi request không hợp lệ.
	ErrInvalidRequest = errors.New("invalid webhook request")
)
