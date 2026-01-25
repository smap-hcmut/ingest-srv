package models

import "time"

// TaskType liệt kê các loại tác vụ crawler hỗ trợ.
type TaskType string

const (
	TaskTypeResearchKeyword  TaskType = "research_keyword"
	TaskTypeCrawlLinks       TaskType = "crawl_links"
	TaskTypeResearchAndCrawl TaskType = "research_and_crawl"
	TaskTypeDryRunKeyword    TaskType = "dryrun_keyword"
)

// Platform liệt kê các worker/platform hiện có.
type Platform string

const (
	PlatformYouTube Platform = "youtube"
	PlatformTikTok  Platform = "tiktok"
)

// BaseCollectorTask chứa các trường chung cho mọi task.
type BaseCollectorTask struct {
	JobID         string         `json:"job_id"`
	Platform      Platform       `json:"platform"`
	TaskType      TaskType       `json:"task_type"`
	TimeRange     int            `json:"time_range,omitempty"`
	Attempt       int            `json:"attempt,omitempty"`
	MaxAttempts   int            `json:"max_attempts,omitempty"`
	Retry         bool           `json:"retry,omitempty"`
	SchemaVersion int            `json:"schema_version,omitempty"`
	TraceID       string         `json:"trace_id,omitempty"`
	RoutingKey    string         `json:"routing_key,omitempty"`
	EmittedAt     time.Time      `json:"emitted_at"`
	Headers       map[string]any `json:"headers,omitempty"`
}

// CollectorTask là interface đánh dấu (marker interface) cho các task.
type CollectorTask interface {
	GetPlatform() Platform
}

func (b BaseCollectorTask) GetPlatform() Platform {
	return b.Platform
}

// TikTokCollectorTask dành riêng cho TikTok.
type TikTokCollectorTask struct {
	BaseCollectorTask
	Payload any `json:"payload"` // Payload cụ thể của TikTok (TikTokResearchKeywordPayload, etc.)
}

// YouTubeCollectorTask dành riêng cho YouTube.
type YouTubeCollectorTask struct {
	BaseCollectorTask
	Payload any `json:"payload"` // Payload cụ thể của YouTube
}

// --- YouTube payloads ---

type YouTubeResearchKeywordPayload struct {
	Keyword   string `json:"keyword"`
	Limit     int    `json:"limit,omitempty"`
	SortBy    string `json:"sort_by,omitempty"`
	TimeRange int    `json:"time_range,omitempty"`
}

type YouTubeCrawlLinksPayload struct {
	VideoURLs       []string `json:"video_urls"`
	IncludeChannel  bool     `json:"include_channel,omitempty"`
	IncludeComments bool     `json:"include_comments,omitempty"`
	MaxComments     int      `json:"max_comments,omitempty"`
	DownloadMedia   bool     `json:"download_media,omitempty"`
	MediaType       string   `json:"media_type,omitempty"`
	TimeRange       int      `json:"time_range,omitempty"`
}

type YouTubeResearchAndCrawlPayload struct {
	Keywords        []string `json:"keywords"`
	LimitPerKeyword int      `json:"limit_per_keyword,omitempty"`
	IncludeComments bool     `json:"include_comments,omitempty"`
	IncludeChannel  bool     `json:"include_channel,omitempty"`
	MaxComments     int      `json:"max_comments,omitempty"`
	DownloadMedia   bool     `json:"download_media,omitempty"`
	TimeRange       int      `json:"time_range,omitempty"`
}

// --- TikTok payloads ---

type TikTokResearchKeywordPayload struct {
	Keyword   string `json:"keyword"`
	Limit     int    `json:"limit,omitempty"`
	SortBy    string `json:"sort_by,omitempty"`
	TimeRange int    `json:"time_range,omitempty"`
}

type TikTokCrawlLinksPayload struct {
	VideoURLs       []string `json:"video_urls"`
	IncludeComments bool     `json:"include_comments,omitempty"`
	IncludeCreator  bool     `json:"include_creator,omitempty"`
	MaxComments     int      `json:"max_comments,omitempty"`
	DownloadMedia   bool     `json:"download_media,omitempty"`
	MediaType       string   `json:"media_type,omitempty"`
	MediaSaveDir    string   `json:"media_save_dir,omitempty"`
	TimeRange       int      `json:"time_range,omitempty"`
}

type TikTokResearchAndCrawlPayload struct {
	Keywords        []string `json:"keywords"`
	LimitPerKeyword int      `json:"limit_per_keyword,omitempty"`
	SortBy          string   `json:"sort_by,omitempty"`
	IncludeComments bool     `json:"include_comments,omitempty"`
	IncludeCreator  bool     `json:"include_creator,omitempty"`
	MaxComments     int      `json:"max_comments,omitempty"`
	DownloadMedia   bool     `json:"download_media,omitempty"`
	MediaType       string   `json:"media_type,omitempty"`
	MediaSaveDir    string   `json:"media_save_dir,omitempty"`
	TimeRange       int      `json:"time_range,omitempty"`
}
