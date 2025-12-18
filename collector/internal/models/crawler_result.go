package models

// CrawlerResult là payload worker gửi ngược collector.
// Supports both legacy format (dry-run with payload) and FLAT format (research_and_crawl).
// FLAT format fields are at root level, legacy format uses Payload array.
type CrawlerResult struct {
	Success bool `json:"success"`
	Payload any  `json:"payload,omitempty"` // Array of content items (legacy/dry-run format)

	// FLAT format fields (research_and_crawl) - Version 3.0
	TaskType        string  `json:"task_type,omitempty"`
	JobID           string  `json:"job_id,omitempty"`
	Platform        string  `json:"platform,omitempty"`
	RequestedLimit  int     `json:"requested_limit,omitempty"`
	AppliedLimit    int     `json:"applied_limit,omitempty"`
	TotalFound      int     `json:"total_found,omitempty"`
	PlatformLimited bool    `json:"platform_limited,omitempty"`
	Successful      int     `json:"successful,omitempty"`
	Failed          int     `json:"failed,omitempty"`
	Skipped         int     `json:"skipped,omitempty"`
	ErrorCode       *string `json:"error_code,omitempty"`
	ErrorMessage    *string `json:"error_message,omitempty"`
}

// ============================================================================
// Crawler Result Message (FLAT format for research_and_crawl)
// Version 3.0: No payload, all fields at root level
// ============================================================================

// CrawlerResultMessage là flat message format từ Crawler cho case research_and_crawl.
// Không có payload - Crawler push content trực tiếp sang Analytics.
type CrawlerResultMessage struct {
	Success         bool    `json:"success"`
	TaskType        string  `json:"task_type"`
	JobID           string  `json:"job_id"`
	Platform        string  `json:"platform"`
	RequestedLimit  int     `json:"requested_limit"`
	AppliedLimit    int     `json:"applied_limit"`
	TotalFound      int     `json:"total_found"`
	PlatformLimited bool    `json:"platform_limited"`
	Successful      int     `json:"successful"`
	Failed          int     `json:"failed"`
	Skipped         int     `json:"skipped"`
	ErrorCode       *string `json:"error_code,omitempty"`
	ErrorMessage    *string `json:"error_message,omitempty"`
}

// ============================================================================
// CrawlerResultMessage Validation Methods
// ============================================================================

// Validate kiểm tra CrawlerResultMessage có hợp lệ không.
func (m *CrawlerResultMessage) Validate() error {
	if m.TaskType == "" {
		return ErrMissingTaskType
	}
	if m.JobID == "" {
		return ErrMissingJobID
	}
	if m.Platform == "" {
		return ErrMissingPlatform
	}
	if m.Successful < 0 || m.Failed < 0 || m.Skipped < 0 {
		return ErrInvalidCounts
	}
	if m.RequestedLimit < 0 || m.AppliedLimit < 0 || m.TotalFound < 0 {
		return ErrInvalidLimits
	}
	return nil
}

// ExtractProjectID extracts project_id from job_id.
// Job ID format: {projectID}-{source}-{index} (e.g., "proj123-brand-0")
func (m *CrawlerResultMessage) ExtractProjectID() string {
	const brandSuffix = "-brand-"
	const competitorSuffix = "-competitor-"

	jobID := m.JobID
	if idx := lastIndex(jobID, brandSuffix); idx != -1 {
		return jobID[:idx]
	}
	if idx := lastIndex(jobID, competitorSuffix); idx != -1 {
		return jobID[:idx]
	}
	return jobID
}

// IsErrorRetryable kiểm tra error có thể retry không.
func (m *CrawlerResultMessage) IsErrorRetryable() bool {
	if m.ErrorCode == nil {
		return false
	}
	switch *m.ErrorCode {
	case "AUTH_FAILED", "INVALID_KEYWORD", "BLOCKED", "RATE_LIMITED_PERMANENT":
		return false
	default:
		return true
	}
}

// lastIndex returns the index of the last occurrence of substr in s, or -1 if not found.
func lastIndex(s, substr string) int {
	for i := len(s) - len(substr); i >= 0; i-- {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// Validation errors
var (
	ErrMissingTaskType = &ValidationError{Field: "task_type", Message: "task_type is required"}
	ErrMissingJobID    = &ValidationError{Field: "job_id", Message: "job_id is required"}
	ErrMissingPlatform = &ValidationError{Field: "platform", Message: "platform is required"}
	ErrInvalidCounts   = &ValidationError{Field: "counts", Message: "successful, failed, skipped must be non-negative"}
	ErrInvalidLimits   = &ValidationError{Field: "limits", Message: "requested_limit, applied_limit, total_found must be non-negative"}
)

// ValidationError represents a validation error for crawler result fields.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return e.Field + ": " + e.Message
}
