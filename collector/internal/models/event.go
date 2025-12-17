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
	return s == ProjectStatusInitializing || s == ProjectStatusProcessing
}

// String trả về string representation của status.
func (s ProjectStatus) String() string {
	return string(s)
}

// ProjectState chứa trạng thái execution của project trong Redis.
// Hybrid state: track cả task-level (completion) và item-level (progress).
type ProjectState struct {
	Status ProjectStatus `json:"status"`

	// Task-level tracking (for completion check)
	// Mỗi response từ Crawler = 1 task
	TasksTotal  int64 `json:"tasks_total"`  // Số tasks dispatch (keywords × platforms)
	TasksDone   int64 `json:"tasks_done"`   // Số tasks hoàn thành
	TasksErrors int64 `json:"tasks_errors"` // Số tasks failed

	// Item-level tracking (for progress display)
	// Số items thực tế crawl được từ platform
	ItemsExpected int64 `json:"items_expected"` // tasks × limit_per_keyword
	ItemsActual   int64 `json:"items_actual"`   // Số items crawl thành công
	ItemsErrors   int64 `json:"items_errors"`   // Số items crawl thất bại

	// Analyze phase counters (unchanged)
	AnalyzeTotal  int64 `json:"analyze_total"`
	AnalyzeDone   int64 `json:"analyze_done"`
	AnalyzeErrors int64 `json:"analyze_errors"`

	// Legacy fields (for backward compatibility)
	// Deprecated: Use TasksTotal, TasksDone, TasksErrors instead
	CrawlTotal  int64 `json:"crawl_total"`
	CrawlDone   int64 `json:"crawl_done"`
	CrawlErrors int64 `json:"crawl_errors"`
}

// IsCrawlComplete kiểm tra crawl phase đã hoàn thành chưa.
// Dựa trên TASK-level để đảm bảo tất cả tasks đã được xử lý.
func (s *ProjectState) IsCrawlComplete() bool {
	// Prefer task-level if available
	if s.TasksTotal > 0 {
		return (s.TasksDone + s.TasksErrors) >= s.TasksTotal
	}
	// Fallback to legacy crawl fields
	return s.CrawlTotal > 0 && (s.CrawlDone+s.CrawlErrors) >= s.CrawlTotal
}

// IsAnalyzeComplete kiểm tra analyze phase đã hoàn thành chưa.
func (s *ProjectState) IsAnalyzeComplete() bool {
	return s.AnalyzeTotal > 0 && (s.AnalyzeDone+s.AnalyzeErrors) >= s.AnalyzeTotal
}

// IsComplete kiểm tra project đã hoàn thành chưa (cả crawl và analyze).
func (s *ProjectState) IsComplete() bool {
	return s.IsCrawlComplete() && s.IsAnalyzeComplete()
}

// CrawlProgressPercent tính phần trăm tiến độ crawl phase.
// Dựa trên ITEM-level để hiển thị chính xác hơn cho user.
// Fallback về task-level nếu items không được track.
func (s *ProjectState) CrawlProgressPercent() float64 {
	// Prefer item-level for more accurate progress
	if s.ItemsExpected > 0 {
		return float64(s.ItemsActual+s.ItemsErrors) / float64(s.ItemsExpected) * 100
	}
	// Fallback to task-level
	if s.TasksTotal > 0 {
		return float64(s.TasksDone+s.TasksErrors) / float64(s.TasksTotal) * 100
	}
	// Legacy fallback
	if s.CrawlTotal > 0 {
		return float64(s.CrawlDone+s.CrawlErrors) / float64(s.CrawlTotal) * 100
	}
	return 0
}

// TasksProgressPercent tính phần trăm tiến độ dựa trên task-level.
func (s *ProjectState) TasksProgressPercent() float64 {
	if s.TasksTotal <= 0 {
		return 0
	}
	return float64(s.TasksDone+s.TasksErrors) / float64(s.TasksTotal) * 100
}

// ItemsProgressPercent tính phần trăm tiến độ dựa trên item-level.
func (s *ProjectState) ItemsProgressPercent() float64 {
	if s.ItemsExpected <= 0 {
		return 0
	}
	return float64(s.ItemsActual+s.ItemsErrors) / float64(s.ItemsExpected) * 100
}

// AnalyzeProgressPercent tính phần trăm tiến độ analyze phase.
func (s *ProjectState) AnalyzeProgressPercent() float64 {
	if s.AnalyzeTotal <= 0 {
		return 0
	}
	return float64(s.AnalyzeDone+s.AnalyzeErrors) / float64(s.AnalyzeTotal) * 100
}

// OverallProgressPercent tính phần trăm tiến độ tổng thể.
// Sử dụng item-level cho crawl progress nếu có, fallback về task-level.
func (s *ProjectState) OverallProgressPercent() float64 {
	crawlProgress := s.CrawlProgressPercent()
	analyzeProgress := s.AnalyzeProgressPercent()
	return (crawlProgress + analyzeProgress) / 2
}

// NewProjectState tạo ProjectState mới với status INITIALIZING.
func NewProjectState() ProjectState {
	return ProjectState{
		Status:        ProjectStatusInitializing,
		TasksTotal:    0,
		TasksDone:     0,
		TasksErrors:   0,
		ItemsExpected: 0,
		ItemsActual:   0,
		ItemsErrors:   0,
		AnalyzeTotal:  0,
		AnalyzeDone:   0,
		AnalyzeErrors: 0,
		// Legacy fields
		CrawlTotal:  0,
		CrawlDone:   0,
		CrawlErrors: 0,
	}
}
