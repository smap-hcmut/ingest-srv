package uap

import (
	"encoding/json"
	"time"
)

type ParseAndStoreRawBatchInput struct {
	RawBatchID      string
	ProjectID       string
	SourceID        string
	ExternalTaskID  string
	TaskID          string
	Platform        string
	Action          string
	StorageBucket   string
	StoragePath     string
	BatchID         string
	RawMetadata     json.RawMessage
	CompletionTime  time.Time
}

type UAPRecord struct {
	Identity   UAPIdentity   `json:"identity"`
	Hierarchy  UAPHierarchy  `json:"hierarchy"`
	Content    UAPContent    `json:"content"`
	Author     UAPAuthor     `json:"author"`
	Engagement UAPEngagement `json:"engagement"`
	Media      []UAPMedia    `json:"media,omitempty"`
	Temporal   UAPTemporal   `json:"temporal"`
}

type UAPIdentity struct {
	UAPID    string `json:"uap_id"`
	OriginID string `json:"origin_id"`
	UAPType  string `json:"uap_type"`
	Platform string `json:"platform"`
	URL      string `json:"url,omitempty"`
	TaskID   string `json:"task_id,omitempty"`
	ProjectID string `json:"project_id,omitempty"`
}

type UAPHierarchy struct {
	ParentID *string `json:"parent_id"`
	RootID   string  `json:"root_id"`
	Depth    int     `json:"depth"`
}

type UAPContent struct {
	Text           string   `json:"text,omitempty"`
	Hashtags       []string `json:"hashtags,omitempty"`
	TikTokKeywords []string `json:"tiktok_keywords,omitempty"`
	IsShopVideo    *bool    `json:"is_shop_video,omitempty"`
	MusicTitle     string   `json:"music_title,omitempty"`
	MusicURL       string   `json:"music_url,omitempty"`
	SummaryTitle   string   `json:"summary_title,omitempty"`
	SubtitleURL    string   `json:"subtitle_url,omitempty"`
	Language       string   `json:"language,omitempty"`
	ExternalLinks  []string `json:"external_links,omitempty"`
}

type UAPAuthor struct {
	ID         string `json:"id,omitempty"`
	Username   string `json:"username,omitempty"`
	Nickname   string `json:"nickname,omitempty"`
	Avatar     string `json:"avatar,omitempty"`
	IsVerified *bool  `json:"is_verified,omitempty"`
}

type UAPEngagement struct {
	Likes         *int     `json:"likes,omitempty"`
	CommentsCount *int     `json:"comments_count,omitempty"`
	Shares        *int     `json:"shares,omitempty"`
	Views         *int     `json:"views,omitempty"`
	Bookmarks     *int     `json:"bookmarks,omitempty"`
	ReplyCount    *int     `json:"reply_count,omitempty"`
	SortScore     *float64 `json:"sort_score,omitempty"`
}

type UAPMedia struct {
	Type        string `json:"type"`
	URL         string `json:"url,omitempty"`
	DownloadURL string `json:"download_url,omitempty"`
	Duration    *int   `json:"duration,omitempty"`
	Thumbnail   string `json:"thumbnail,omitempty"`
}

type UAPTemporal struct {
	PostedAt   string `json:"posted_at,omitempty"`
	IngestedAt string `json:"ingested_at,omitempty"`
}
