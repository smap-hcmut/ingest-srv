package results

// CrawlerPayload represents the complete payload from crawler with success status and content array
type CrawlerPayload struct {
	Success bool             `json:"success"`
	Payload []CrawlerContent `json:"payload"`
}

// CrawlerContent represents a single content item from crawler with all nested structures
type CrawlerContent struct {
	Meta        CrawlerContentMeta        `json:"meta"`
	Content     CrawlerContentData        `json:"content"`
	Interaction CrawlerContentInteraction `json:"interaction"`
	Author      CrawlerContentAuthor      `json:"author"`
	Comments    []CrawlerComment          `json:"comments,omitempty"`
}

// CrawlerContentMeta contains metadata about the content including identifiers and timestamps
type CrawlerContentMeta struct {
	ID              string  `json:"id"`
	Platform        string  `json:"platform"`
	JobID           string  `json:"job_id"`
	TaskType        string  `json:"task_type,omitempty"` // "dryrun_keyword" or "research_and_crawl"
	CrawledAt       string  `json:"crawled_at"`          // RFC3339 string
	PublishedAt     string  `json:"published_at"`        // RFC3339 string
	Permalink       string  `json:"permalink"`
	KeywordSource   string  `json:"keyword_source"`
	Lang            string  `json:"lang"`
	Region          string  `json:"region"`
	PipelineVersion string  `json:"pipeline_version"`
	FetchStatus     string  `json:"fetch_status"`
	FetchError      *string `json:"fetch_error,omitempty"`
}

// CrawlerContentData contains the content data including text, media, and metadata
type CrawlerContentData struct {
	Text          string               `json:"text"`
	Duration      int                  `json:"duration,omitempty"`
	Hashtags      []string             `json:"hashtags,omitempty"`
	SoundName     string               `json:"sound_name,omitempty"`
	Category      *string              `json:"category,omitempty"`
	Title         *string              `json:"title,omitempty"` // YouTube only
	Media         *CrawlerContentMedia `json:"media,omitempty"`
	Transcription *string              `json:"transcription,omitempty"`
}

// CrawlerContentMedia contains media information including paths and download timestamp
type CrawlerContentMedia struct {
	Type         string `json:"type"`
	VideoPath    string `json:"video_path,omitempty"`
	AudioPath    string `json:"audio_path,omitempty"`
	DownloadedAt string `json:"downloaded_at,omitempty"` // RFC3339 string
}

// CrawlerContentInteraction contains engagement metrics for the content
type CrawlerContentInteraction struct {
	Views          int     `json:"views"`
	Likes          int     `json:"likes"`
	CommentsCount  int     `json:"comments_count"`
	Shares         int     `json:"shares"`
	Saves          int     `json:"saves,omitempty"`
	EngagementRate float64 `json:"engagement_rate,omitempty"`
	UpdatedAt      string  `json:"updated_at"` // RFC3339 string
}

// CrawlerContentAuthor contains author information including profile details and statistics
type CrawlerContentAuthor struct {
	ID             string  `json:"id"`
	Name           string  `json:"name"`
	Username       string  `json:"username"`
	Followers      int     `json:"followers"`
	Following      int     `json:"following"`
	Likes          int     `json:"likes"`
	Videos         int     `json:"videos"`
	IsVerified     bool    `json:"is_verified"`
	Bio            string  `json:"bio,omitempty"`
	AvatarURL      *string `json:"avatar_url,omitempty"`
	ProfileURL     string  `json:"profile_url"`
	Country        *string `json:"country,omitempty"`          // YouTube only
	TotalViewCount *int    `json:"total_view_count,omitempty"` // YouTube only
}

// CrawlerComment represents a comment on a post with user information and metadata
type CrawlerComment struct {
	ID           string             `json:"id"`
	ParentID     *string            `json:"parent_id,omitempty"`
	PostID       string             `json:"post_id"`
	User         CrawlerCommentUser `json:"user"`
	Text         string             `json:"text"`
	Likes        int                `json:"likes"`
	RepliesCount int                `json:"replies_count"`
	PublishedAt  string             `json:"published_at"` // RFC3339 string
	IsAuthor     bool               `json:"is_author"`
	Media        *string            `json:"media,omitempty"`
	IsFavorited  bool               `json:"is_favorited"` // YouTube only
}

// CrawlerCommentUser contains comment author information
type CrawlerCommentUser struct {
	ID        *string `json:"id,omitempty"`
	Name      string  `json:"name"`
	AvatarURL *string `json:"avatar_url,omitempty"`
}

// TaskType constants for routing messages
const (
	// TaskTypeAnalyzeResult is the required task_type for Analytics → Collector messages
	TaskTypeAnalyzeResult = "analyze_result"
	// TaskTypeResearchAndCrawl is the task_type for production crawl tasks
	TaskTypeResearchAndCrawl = "research_and_crawl"
	// TaskTypeDryrunKeyword is the task_type for dry-run/preview tasks
	TaskTypeDryrunKeyword = "dryrun_keyword"
)

// AnalyzeResultPayload represents the payload from Analytics Service
// when an analyze batch completes.
type AnalyzeResultPayload struct {
	ProjectID    string `json:"project_id"`
	JobID        string `json:"job_id"`
	TaskType     string `json:"task_type"` // Must be "analyze_result"
	BatchSize    int    `json:"batch_size"`
	SuccessCount int    `json:"success_count"`
	ErrorCount   int    `json:"error_count"`
}

// Validate kiểm tra AnalyzeResultPayload có hợp lệ không.
// Returns error nếu có vấn đề với data.
func (p *AnalyzeResultPayload) Validate() error {
	if p.TaskType != TaskTypeAnalyzeResult {
		return ErrInvalidTaskType
	}
	if p.ProjectID == "" {
		return ErrMissingProjectID
	}
	if p.SuccessCount < 0 || p.ErrorCount < 0 || p.BatchSize < 0 {
		return ErrInvalidAnalyzeCounts
	}
	return nil
}

// IsValid kiểm tra payload có hợp lệ để xử lý không.
// Shorthand for Validate() == nil.
func (p *AnalyzeResultPayload) IsValid() bool {
	return p.Validate() == nil
}

// TotalProcessed trả về tổng số items đã xử lý (success + error).
func (p *AnalyzeResultPayload) TotalProcessed() int {
	return p.SuccessCount + p.ErrorCount
}

// Validation errors for AnalyzeResultPayload
var (
	ErrInvalidTaskType      = &AnalyzeValidationError{Field: "task_type", Message: "task_type must be 'analyze_result'"}
	ErrMissingProjectID     = &AnalyzeValidationError{Field: "project_id", Message: "project_id is required"}
	ErrInvalidAnalyzeCounts = &AnalyzeValidationError{Field: "counts", Message: "success_count, error_count, and batch_size must be non-negative"}
)

// AnalyzeValidationError represents a validation error for analyze result payload.
type AnalyzeValidationError struct {
	Field   string
	Message string
}

func (e *AnalyzeValidationError) Error() string {
	return e.Field + ": " + e.Message
}
