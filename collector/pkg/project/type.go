package project

import "time"

// CallbackRequest represents the webhook callback payload sent to Project Service
type CallbackRequest struct {
	JobID    string          `json:"job_id"`
	Status   string          `json:"status"` // "success" or "failed"
	Platform string          `json:"platform"`
	Payload  CallbackPayload `json:"payload"`
}

// CallbackPayload contains the crawl results or errors.
// The Content field is named to maintain consistency with the crawler's data structure
// where the payload contains content items rather than posts.
type CallbackPayload struct {
	Content []Content `json:"content,omitempty"` // Array of crawled content items
	Errors  []Error   `json:"errors,omitempty"`
}

// Content represents a single social media content item with full crawler data
type Content struct {
	Meta        ContentMeta        `json:"meta"`
	Content     ContentData        `json:"content"`
	Interaction ContentInteraction `json:"interaction"`
	Author      ContentAuthor      `json:"author"`
	Comments    []Comment          `json:"comments,omitempty"`
}

// ContentMeta contains metadata about the content
type ContentMeta struct {
	ID              string    `json:"id"`
	Platform        string    `json:"platform"`
	JobID           string    `json:"job_id"`
	CrawledAt       time.Time `json:"crawled_at"`
	PublishedAt     time.Time `json:"published_at"`
	Permalink       string    `json:"permalink"`
	KeywordSource   string    `json:"keyword_source"`
	Lang            string    `json:"lang"`
	Region          string    `json:"region"`
	PipelineVersion string    `json:"pipeline_version"`
	FetchStatus     string    `json:"fetch_status"`
	FetchError      *string   `json:"fetch_error"`
}

// ContentData contains the content data
type ContentData struct {
	Text          string        `json:"text"`
	Duration      int           `json:"duration,omitempty"`
	Hashtags      []string      `json:"hashtags,omitempty"`
	SoundName     string        `json:"sound_name,omitempty"`
	Category      *string       `json:"category,omitempty"`
	Title         *string       `json:"title,omitempty"` // YouTube only
	Media         *ContentMedia `json:"media,omitempty"`
	Transcription *string       `json:"transcription,omitempty"`
}

// ContentMedia contains media information
type ContentMedia struct {
	Type         string    `json:"type"`
	VideoPath    string    `json:"video_path,omitempty"`
	AudioPath    string    `json:"audio_path,omitempty"`
	DownloadedAt time.Time `json:"downloaded_at,omitempty"`
}

// ContentInteraction contains engagement metrics
type ContentInteraction struct {
	Views          int       `json:"views"`
	Likes          int       `json:"likes"`
	CommentsCount  int       `json:"comments_count"`
	Shares         int       `json:"shares"`
	Saves          int       `json:"saves,omitempty"`
	EngagementRate float64   `json:"engagement_rate,omitempty"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// ContentAuthor contains author information
type ContentAuthor struct {
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

// Comment represents a comment on a post
type Comment struct {
	ID           string      `json:"id"`
	ParentID     *string     `json:"parent_id,omitempty"`
	PostID       string      `json:"post_id"`
	User         CommentUser `json:"user"`
	Text         string      `json:"text"`
	Likes        int         `json:"likes"`
	RepliesCount int         `json:"replies_count"`
	PublishedAt  time.Time   `json:"published_at"`
	IsAuthor     bool        `json:"is_author"`
	Media        *string     `json:"media,omitempty"`
	IsFavorited  bool        `json:"is_favorited"` // YouTube only
}

// CommentUser contains comment author information
type CommentUser struct {
	ID        *string `json:"id,omitempty"`
	Name      string  `json:"name"`
	AvatarURL *string `json:"avatar_url,omitempty"`
}

// Error represents a crawl error
type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Keyword string `json:"keyword,omitempty"`
}

// ProgressCallbackRequest represents the progress webhook callback payload.
// Gửi tới POST /internal/progress/callback
type ProgressCallbackRequest struct {
	ProjectID string `json:"project_id"`
	UserID    string `json:"user_id"`
	Status    string `json:"status"` // INITIALIZING, CRAWLING, PROCESSING, DONE, FAILED
	Total     int64  `json:"total"`
	Done      int64  `json:"done"`
	Errors    int64  `json:"errors"`
}
