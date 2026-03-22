package kafka

import (
	"encoding/json"

	"ingest-srv/internal/uap"
)

type uapRecordPayload struct {
	Identity   uapIdentityPayload   `json:"identity"`
	Hierarchy  uapHierarchyPayload  `json:"hierarchy"`
	Content    uapContentPayload    `json:"content"`
	Author     uapAuthorPayload     `json:"author"`
	Engagement uapEngagementPayload `json:"engagement"`
	Media      []uapMediaPayload    `json:"media,omitempty"`
	Temporal   uapTemporalPayload   `json:"temporal"`
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

type uapAuthorPayload struct {
	ID         string `json:"id,omitempty"`
	Username   string `json:"username,omitempty"`
	Nickname   string `json:"nickname,omitempty"`
	Avatar     string `json:"avatar,omitempty"`
	IsVerified *bool  `json:"is_verified,omitempty"`
}

type uapEngagementPayload struct {
	Likes         *int     `json:"likes,omitempty"`
	CommentsCount *int     `json:"comments_count,omitempty"`
	Shares        *int     `json:"shares,omitempty"`
	Views         *int     `json:"views,omitempty"`
	Bookmarks     *int     `json:"bookmarks,omitempty"`
	ReplyCount    *int     `json:"reply_count,omitempty"`
	SortScore     *float64 `json:"sort_score,omitempty"`
}

type uapMediaPayload struct {
	Type        string `json:"type"`
	URL         string `json:"url,omitempty"`
	DownloadURL string `json:"download_url,omitempty"`
	Duration    *int   `json:"duration,omitempty"`
	Thumbnail   string `json:"thumbnail,omitempty"`
}

type uapTemporalPayload struct {
	PostedAt   string `json:"posted_at,omitempty"`
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
			Text:           input.Content.Text,
			Hashtags:       input.Content.Hashtags,
			TikTokKeywords: input.Content.TikTokKeywords,
			IsShopVideo:    input.Content.IsShopVideo,
			MusicTitle:     input.Content.MusicTitle,
			MusicURL:       input.Content.MusicURL,
			SummaryTitle:   input.Content.SummaryTitle,
			SubtitleURL:    input.Content.SubtitleURL,
			Language:       input.Content.Language,
			ExternalLinks:  input.Content.ExternalLinks,
		},
		Author: uapAuthorPayload{
			ID:         input.Author.ID,
			Username:   input.Author.Username,
			Nickname:   input.Author.Nickname,
			Avatar:     input.Author.Avatar,
			IsVerified: input.Author.IsVerified,
		},
		Engagement: uapEngagementPayload{
			Likes:         input.Engagement.Likes,
			CommentsCount: input.Engagement.CommentsCount,
			Shares:        input.Engagement.Shares,
			Views:         input.Engagement.Views,
			Bookmarks:     input.Engagement.Bookmarks,
			ReplyCount:    input.Engagement.ReplyCount,
			SortScore:     input.Engagement.SortScore,
		},
		Media: media,
		Temporal: uapTemporalPayload{
			PostedAt:   input.Temporal.PostedAt,
			IngestedAt: input.Temporal.IngestedAt,
		},
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
			Text:           input.Content.Text,
			Hashtags:       input.Content.Hashtags,
			TikTokKeywords: input.Content.TikTokKeywords,
			IsShopVideo:    input.Content.IsShopVideo,
			MusicTitle:     input.Content.MusicTitle,
			MusicURL:       input.Content.MusicURL,
			SummaryTitle:   input.Content.SummaryTitle,
			SubtitleURL:    input.Content.SubtitleURL,
			Language:       input.Content.Language,
			ExternalLinks:  input.Content.ExternalLinks,
		},
		Author: uap.UAPAuthor{
			ID:         input.Author.ID,
			Username:   input.Author.Username,
			Nickname:   input.Author.Nickname,
			Avatar:     input.Author.Avatar,
			IsVerified: input.Author.IsVerified,
		},
		Engagement: uap.UAPEngagement{
			Likes:         input.Engagement.Likes,
			CommentsCount: input.Engagement.CommentsCount,
			Shares:        input.Engagement.Shares,
			Views:         input.Engagement.Views,
			Bookmarks:     input.Engagement.Bookmarks,
			ReplyCount:    input.Engagement.ReplyCount,
			SortScore:     input.Engagement.SortScore,
		},
		Media: media,
		Temporal: uap.UAPTemporal{
			PostedAt:   input.Temporal.PostedAt,
			IngestedAt: input.Temporal.IngestedAt,
		},
	}
}
