package kafka

import (
	"encoding/json"

	"ingest-srv/internal/uap"
)

type uapRecordPayload struct {
	Identity     uapIdentityPayload     `json:"identity"`
	Hierarchy    uapHierarchyPayload    `json:"hierarchy"`
	Content      uapContentPayload      `json:"content"`
	Author       uapAuthorPayload       `json:"author"`
	Engagement   uapEngagementPayload   `json:"engagement"`
	Media        []uapMediaPayload      `json:"media,omitempty"`
	Temporal     uapTemporalPayload     `json:"temporal"`
	CrawlKeyword string                 `json:"crawl_keyword,omitempty"`
	PlatformMeta map[string]interface{} `json:"platform_meta,omitempty"`
}

type uapIdentityPayload struct {
	UAPID     string `json:"uap_id"`
	OriginID  string `json:"origin_id"`
	UAPType   string `json:"uap_type"`
	Platform  string `json:"platform"`
	URL       string `json:"url,omitempty"`
	TaskID    string `json:"task_id,omitempty"`
	ProjectID string `json:"project_id,omitempty"`
}

type uapHierarchyPayload struct {
	ParentID *string `json:"parent_id"`
	RootID   string  `json:"root_id"`
	Depth    int     `json:"depth"`
}

type uapContentPayload struct {
	Text     string   `json:"text,omitempty"`
	Title    string   `json:"title,omitempty"`
	Subtitle string   `json:"subtitle,omitempty"`
	Hashtags []string `json:"hashtags,omitempty"`
	Keywords []string `json:"keywords,omitempty"`
	Language string   `json:"language,omitempty"`
	Links    []string `json:"links,omitempty"`
}

type uapAuthorPayload struct {
	ID         string `json:"id,omitempty"`
	Username   string `json:"username,omitempty"`
	Nickname   string `json:"nickname,omitempty"`
	Avatar     string `json:"avatar,omitempty"`
	ProfileURL string `json:"profile_url,omitempty"`
	IsVerified *bool  `json:"is_verified,omitempty"`
}

type uapEngagementPayload struct {
	Likes         *int `json:"likes,omitempty"`
	CommentsCount *int `json:"comments_count,omitempty"`
	Shares        *int `json:"shares,omitempty"`
	Views         *int `json:"views,omitempty"`
	Saves         *int `json:"saves,omitempty"`
	ReplyCount    *int `json:"reply_count,omitempty"`
}

type uapMediaPayload struct {
	Type        string `json:"type"`
	URL         string `json:"url,omitempty"`
	DownloadURL string `json:"download_url,omitempty"`
	Duration    *int   `json:"duration,omitempty"`
	Thumbnail   string `json:"thumbnail,omitempty"`
	Width       *int   `json:"width,omitempty"`
	Height      *int   `json:"height,omitempty"`
}

type uapTemporalPayload struct {
	PostedAt   string `json:"posted_at,omitempty"`
	UpdatedAt  string `json:"updated_at,omitempty"`
	IngestedAt string `json:"ingested_at,omitempty"`
}

func MarshalUAPRecord(input uap.UAPRecord) ([]byte, error) {
	return json.Marshal(toPayload(input))
}

func UnmarshalUAPRecord(raw []byte) (uap.UAPRecord, error) {
	var payload uapRecordPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return uap.UAPRecord{}, err
	}
	return fromPayload(payload), nil
}

func toPayload(input uap.UAPRecord) uapRecordPayload {
	media := make([]uapMediaPayload, 0, len(input.Media))
	for _, item := range input.Media {
		media = append(media, uapMediaPayload{
			Type:        item.Type,
			URL:         item.URL,
			DownloadURL: item.DownloadURL,
			Duration:    item.Duration,
			Thumbnail:   item.Thumbnail,
			Width:       item.Width,
			Height:      item.Height,
		})
	}

	return uapRecordPayload{
		Identity: uapIdentityPayload{
			UAPID:     input.Identity.UAPID,
			OriginID:  input.Identity.OriginID,
			UAPType:   string(input.Identity.UAPType),
			Platform:  input.Identity.Platform,
			URL:       input.Identity.URL,
			TaskID:    input.Identity.TaskID,
			ProjectID: input.Identity.ProjectID,
		},
		Hierarchy: uapHierarchyPayload{
			ParentID: input.Hierarchy.ParentID,
			RootID:   input.Hierarchy.RootID,
			Depth:    input.Hierarchy.Depth,
		},
		Content: uapContentPayload{
			Text:     input.Content.Text,
			Title:    input.Content.Title,
			Subtitle: input.Content.Subtitle,
			Hashtags: input.Content.Hashtags,
			Keywords: input.Content.Keywords,
			Language: input.Content.Language,
			Links:    input.Content.Links,
		},
		Author: uapAuthorPayload{
			ID:         input.Author.ID,
			Username:   input.Author.Username,
			Nickname:   input.Author.Nickname,
			Avatar:     input.Author.Avatar,
			ProfileURL: input.Author.ProfileURL,
			IsVerified: input.Author.IsVerified,
		},
		Engagement: uapEngagementPayload{
			Likes:         input.Engagement.Likes,
			CommentsCount: input.Engagement.CommentsCount,
			Shares:        input.Engagement.Shares,
			Views:         input.Engagement.Views,
			Saves:         input.Engagement.Saves,
			ReplyCount:    input.Engagement.ReplyCount,
		},
		Media: media,
		Temporal: uapTemporalPayload{
			PostedAt:   input.Temporal.PostedAt,
			UpdatedAt:  input.Temporal.UpdatedAt,
			IngestedAt: input.Temporal.IngestedAt,
		},
		CrawlKeyword: input.CrawlKeyword,
		PlatformMeta: input.PlatformMeta,
	}
}

func fromPayload(input uapRecordPayload) uap.UAPRecord {
	media := make([]uap.UAPMedia, 0, len(input.Media))
	for _, item := range input.Media {
		media = append(media, uap.UAPMedia{
			Type:        item.Type,
			URL:         item.URL,
			DownloadURL: item.DownloadURL,
			Duration:    item.Duration,
			Thumbnail:   item.Thumbnail,
			Width:       item.Width,
			Height:      item.Height,
		})
	}

	return uap.UAPRecord{
		Identity: uap.UAPIdentity{
			UAPID:     input.Identity.UAPID,
			OriginID:  input.Identity.OriginID,
			UAPType:   uap.UAPType(input.Identity.UAPType),
			Platform:  input.Identity.Platform,
			URL:       input.Identity.URL,
			TaskID:    input.Identity.TaskID,
			ProjectID: input.Identity.ProjectID,
		},
		Hierarchy: uap.UAPHierarchy{
			ParentID: input.Hierarchy.ParentID,
			RootID:   input.Hierarchy.RootID,
			Depth:    input.Hierarchy.Depth,
		},
		Content: uap.UAPContent{
			Text:     input.Content.Text,
			Title:    input.Content.Title,
			Subtitle: input.Content.Subtitle,
			Hashtags: input.Content.Hashtags,
			Keywords: input.Content.Keywords,
			Language: input.Content.Language,
			Links:    input.Content.Links,
		},
		Author: uap.UAPAuthor{
			ID:         input.Author.ID,
			Username:   input.Author.Username,
			Nickname:   input.Author.Nickname,
			Avatar:     input.Author.Avatar,
			ProfileURL: input.Author.ProfileURL,
			IsVerified: input.Author.IsVerified,
		},
		Engagement: uap.UAPEngagement{
			Likes:         input.Engagement.Likes,
			CommentsCount: input.Engagement.CommentsCount,
			Shares:        input.Engagement.Shares,
			Views:         input.Engagement.Views,
			Saves:         input.Engagement.Saves,
			ReplyCount:    input.Engagement.ReplyCount,
		},
		Media: media,
		Temporal: uap.UAPTemporal{
			PostedAt:   input.Temporal.PostedAt,
			UpdatedAt:  input.Temporal.UpdatedAt,
			IngestedAt: input.Temporal.IngestedAt,
		},
		CrawlKeyword: input.CrawlKeyword,
		PlatformMeta: input.PlatformMeta,
	}
}
