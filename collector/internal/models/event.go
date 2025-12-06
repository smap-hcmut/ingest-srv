package models

import "time"

// ============================================================================
// Project Created Event (from Project Service)
// ============================================================================

// ProjectCreatedEvent là event được publish bởi Project Service khi user
// execute một project. Collector Service consume event này để bắt đầu crawling.
type ProjectCreatedEvent struct {
	EventID   string                `json:"event_id"`
	Timestamp time.Time             `json:"timestamp"`
	Payload   ProjectCreatedPayload `json:"payload"`
}

// ProjectCreatedPayload chứa thông tin chi tiết về project cần crawl.
type ProjectCreatedPayload struct {
	ProjectID             string              `json:"project_id"`
	UserID                string              `json:"user_id"` // For progress notifications
	BrandName             string              `json:"brand_name"`
	BrandKeywords         []string            `json:"brand_keywords"`
	CompetitorNames       []string            `json:"competitor_names"`
	CompetitorKeywordsMap map[string][]string `json:"competitor_keywords_map"`
	DateRange             DateRange           `json:"date_range"`
}

// DateRange định nghĩa khoảng thời gian cần crawl data.
type DateRange struct {
	From string `json:"from"` // Format: YYYY-MM-DD
	To   string `json:"to"`   // Format: YYYY-MM-DD
}

// IsValid kiểm tra event có hợp lệ không.
func (e *ProjectCreatedEvent) IsValid() bool {
	return e.EventID != "" &&
		e.Payload.ProjectID != "" &&
		e.Payload.UserID != ""
}

// GetProjectID trả về project ID từ event.
func (e *ProjectCreatedEvent) GetProjectID() string {
	return e.Payload.ProjectID
}

// GetUserID trả về user ID từ event.
func (e *ProjectCreatedEvent) GetUserID() string {
	return e.Payload.UserID
}

// GetAllKeywords trả về tất cả keywords (brand + competitors) cần crawl.
func (e *ProjectCreatedEvent) GetAllKeywords() []string {
	keywords := make([]string, 0)

	// Add brand keywords
	keywords = append(keywords, e.Payload.BrandKeywords...)

	// Add competitor keywords
	for _, competitorKeywords := range e.Payload.CompetitorKeywordsMap {
		keywords = append(keywords, competitorKeywords...)
	}

	return keywords
}

// ============================================================================
// Note: DataCollectedEvent is published by Crawler (Worker) services, not Collector.
// See document/event-drivent.md for the correct event flow.
// ============================================================================

// ============================================================================
// Project Status & State (for Redis state management)
// ============================================================================

// ProjectStatus định nghĩa các trạng thái của project execution.
type ProjectStatus string

const (
	ProjectStatusInitializing ProjectStatus = "INITIALIZING"
	ProjectStatusCrawling     ProjectStatus = "CRAWLING"
	ProjectStatusProcessing   ProjectStatus = "PROCESSING"
	ProjectStatusDone         ProjectStatus = "DONE"
	ProjectStatusFailed       ProjectStatus = "FAILED"
)

// IsTerminal kiểm tra status có phải là trạng thái kết thúc không.
func (s ProjectStatus) IsTerminal() bool {
	return s == ProjectStatusDone || s == ProjectStatusFailed
}

// IsActive kiểm tra status có phải là trạng thái đang chạy không.
func (s ProjectStatus) IsActive() bool {
	return s == ProjectStatusInitializing ||
		s == ProjectStatusCrawling ||
		s == ProjectStatusProcessing
}

// String trả về string representation của status.
func (s ProjectStatus) String() string {
	return string(s)
}

// ProjectState chứa trạng thái execution của project trong Redis.
type ProjectState struct {
	Status ProjectStatus `json:"status"`
	Total  int64         `json:"total"`
	Done   int64         `json:"done"`
	Errors int64         `json:"errors"`
}

// IsComplete kiểm tra project đã hoàn thành chưa (done + errors >= total).
func (s *ProjectState) IsComplete() bool {
	return s.Total > 0 && (s.Done+s.Errors) >= s.Total
}

// ProgressPercent tính phần trăm tiến độ.
func (s *ProjectState) ProgressPercent() float64 {
	if s.Total <= 0 {
		return 0
	}
	return float64(s.Done) / float64(s.Total) * 100
}

// NewProjectState tạo ProjectState mới với status INITIALIZING.
func NewProjectState() ProjectState {
	return ProjectState{
		Status: ProjectStatusInitializing,
		Total:  0,
		Done:   0,
		Errors: 0,
	}
}
