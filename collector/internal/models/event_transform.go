package models

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// TransformProjectEventToRequests chuyển đổi ProjectCreatedEvent thành
// danh sách CrawlRequest để dispatch tới các workers.
//
// Logic:
// - Mỗi keyword (brand + competitor) sẽ tạo ra 1 CrawlRequest
// - TaskType là research_and_crawl để vừa search vừa crawl
// - JobID format: {projectID}-{source}-{keyword_index}
func TransformProjectEventToRequests(event ProjectCreatedEvent, opts TransformOptions) []CrawlRequest {
	requests := make([]CrawlRequest, 0)

	// Calculate time range in days from DateRange
	timeRange := calculateTimeRange(event.Payload.DateRange)

	// Transform brand keywords
	for i, keyword := range event.Payload.BrandKeywords {
		req := CrawlRequest{
			JobID:       fmt.Sprintf("%s-brand-%d", event.Payload.ProjectID, i),
			TaskType:    TaskTypeResearchAndCrawl,
			TimeRange:   timeRange,
			Attempt:     1,
			MaxAttempts: opts.MaxAttempts,
			EmittedAt:   time.Now().UTC(),
			Payload: map[string]any{
				"keywords":          []string{keyword},
				"limit_per_keyword": opts.LimitPerKeyword,
				"include_comments":  opts.IncludeComments,
				"max_comments":      opts.MaxComments,
				"download_media":    opts.DownloadMedia,
				"time_range":        timeRange,
				// Metadata for tracking
				"project_id":     event.Payload.ProjectID,
				"user_id":        event.Payload.UserID,
				"keyword_source": "brand",
				"brand_name":     event.Payload.BrandName,
			},
		}
		requests = append(requests, req)
	}

	// Transform competitor keywords
	for competitor, keywords := range event.Payload.CompetitorKeywordsMap {
		for i, keyword := range keywords {
			req := CrawlRequest{
				JobID:       fmt.Sprintf("%s-competitor-%s-%d", event.Payload.ProjectID, competitor, i),
				TaskType:    TaskTypeResearchAndCrawl,
				TimeRange:   timeRange,
				Attempt:     1,
				MaxAttempts: opts.MaxAttempts,
				EmittedAt:   time.Now().UTC(),
				Payload: map[string]any{
					"keywords":          []string{keyword},
					"limit_per_keyword": opts.LimitPerKeyword,
					"include_comments":  opts.IncludeComments,
					"max_comments":      opts.MaxComments,
					"download_media":    opts.DownloadMedia,
					"time_range":        timeRange,
					// Metadata for tracking
					"project_id":      event.Payload.ProjectID,
					"user_id":         event.Payload.UserID,
					"keyword_source":  "competitor",
					"competitor_name": competitor,
				},
			}
			requests = append(requests, req)
		}
	}

	return requests
}

// TransformOptions chứa các tùy chọn cho việc transform event.
type TransformOptions struct {
	MaxAttempts     int  // Default: 3
	LimitPerKeyword int  // Default: 50
	IncludeComments bool // Default: true
	MaxComments     int  // Default: 100
	DownloadMedia   bool // Default: false
}

// DefaultTransformOptions trả về options mặc định.
func DefaultTransformOptions() TransformOptions {
	return TransformOptions{
		MaxAttempts:     3,
		LimitPerKeyword: 50,
		IncludeComments: true,
		MaxComments:     100,
		DownloadMedia:   false,
	}
}

// calculateTimeRange tính số ngày từ DateRange.
// Trả về số ngày giữa From và To, hoặc 30 nếu không parse được.
func calculateTimeRange(dr DateRange) int {
	if dr.From == "" || dr.To == "" {
		return 30 // Default 30 days
	}

	from, err := time.Parse("2006-01-02", dr.From)
	if err != nil {
		return 30
	}

	to, err := time.Parse("2006-01-02", dr.To)
	if err != nil {
		return 30
	}

	days := int(to.Sub(from).Hours() / 24)
	if days <= 0 {
		return 30
	}

	return days
}

// CountTotalTasks đếm tổng số tasks sẽ được tạo từ event.
// Mỗi keyword = 1 task * số platforms (YouTube + TikTok = 2)
func CountTotalTasks(event ProjectCreatedEvent, platformCount int) int {
	keywordCount := len(event.Payload.BrandKeywords)

	for _, keywords := range event.Payload.CompetitorKeywordsMap {
		keywordCount += len(keywords)
	}

	return keywordCount * platformCount
}

// GenerateEventID tạo event ID mới với prefix.
func GenerateEventID(prefix string) string {
	return fmt.Sprintf("%s_%s", prefix, uuid.New().String()[:8])
}

// ProjectUserMapping lưu mapping giữa project_id và user_id.
// Dùng để lookup user_id khi cần notify progress.
type ProjectUserMapping struct {
	ProjectID string
	UserID    string
	CreatedAt time.Time
}

// NewProjectUserMapping tạo mapping mới từ event.
func NewProjectUserMapping(event ProjectCreatedEvent) ProjectUserMapping {
	return ProjectUserMapping{
		ProjectID: event.Payload.ProjectID,
		UserID:    event.Payload.UserID,
		CreatedAt: time.Now().UTC(),
	}
}
