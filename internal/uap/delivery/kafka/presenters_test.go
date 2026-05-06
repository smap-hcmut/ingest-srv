package kafka

import (
	"testing"

	"ingest-srv/internal/uap"

	"github.com/stretchr/testify/require"
)

func TestMarshalUAPRecord(t *testing.T) {
	parentID := "parent-1"
	verified := true
	likes := 10
	duration := 30

	tcs := map[string]struct {
		input  uap.UAPRecord
		mock   struct{}
		output uap.UAPRecord
		err    error
	}{
		"success": {
			input: uap.UAPRecord{
				Identity:  uap.UAPIdentity{UAPID: "uap-1", OriginID: "origin-1", UAPType: uap.UAPTypePost, Platform: "youtube", URL: "https://example.com", TaskID: "task-1", ProjectID: "project-1"},
				Hierarchy: uap.UAPHierarchy{ParentID: &parentID, RootID: "root-1", Depth: 1},
				Content:   uap.UAPContent{Text: "text", Title: "title", Subtitle: "subtitle", Hashtags: []string{"a"}, Keywords: []string{"k"}, Language: "vi", Links: []string{"https://example.com"}},
				Author:    uap.UAPAuthor{ID: "author-1", Username: "user", Nickname: "nick", Avatar: "avatar", ProfileURL: "profile", IsVerified: &verified},
				Engagement: uap.UAPEngagement{
					Likes: &likes,
				},
				Media:          []uap.UAPMedia{{Type: "video", URL: "url", DownloadURL: "download", Duration: &duration, Thumbnail: "thumb"}},
				Temporal:       uap.UAPTemporal{PostedAt: "posted", UpdatedAt: "updated", IngestedAt: "ingested"},
				DomainTypeCode: "generic",
				CrawlKeyword:   "keyword",
				PlatformMeta:   map[string]interface{}{"source": "test"},
			},
		},
	}
	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			body, err := MarshalUAPRecord(tc.input)

			require.ErrorIs(t, err, tc.err)
			require.NotEmpty(t, body)

			got, err := UnmarshalUAPRecord(body)

			require.NoError(t, err)
			require.Equal(t, tc.input.Identity, got.Identity)
			require.Equal(t, tc.input.Hierarchy, got.Hierarchy)
			require.Equal(t, tc.input.Content, got.Content)
			require.Equal(t, tc.input.Author, got.Author)
			require.Equal(t, tc.input.Engagement.Likes, got.Engagement.Likes)
			require.Equal(t, tc.input.Media, got.Media)
			require.Equal(t, tc.input.Temporal, got.Temporal)
			require.Equal(t, tc.input.DomainTypeCode, got.DomainTypeCode)
			require.Equal(t, tc.input.CrawlKeyword, got.CrawlKeyword)
		})
	}
}

func TestUnmarshalUAPRecord(t *testing.T) {
	tcs := map[string]struct {
		input  []byte
		mock   struct{}
		output uap.UAPRecord
		err    bool
	}{
		"invalid": {input: []byte(`{`), err: true},
	}
	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			_, err := UnmarshalUAPRecord(tc.input)
			require.Equal(t, tc.err, err != nil)
		})
	}
}
