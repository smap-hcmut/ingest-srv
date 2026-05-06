package usecase

import (
	"testing"
	"time"

	"ingest-srv/internal/uap"

	"github.com/stretchr/testify/require"
)

func TestFacebookParserEdges(t *testing.T) {
	uc := &implUseCase{}
	input := uap.ParseAndStoreRawBatchInput{TaskID: "task-1", ProjectID: "project-1", CompletionTime: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}

	tcs := map[string]struct {
		input func()
		err   error
	}{
		"flatten invalid json": {
			input: func() {
				_, err := uc.flattenFacebookFullFlow([]byte(`{`), input, nil)
				require.ErrorIs(t, err, uap.ErrParseRawPayload)
			},
		},
		"parse skips empty post map": {
			input: func() {
				parsed, err := uc.parseFacebookFullFlowInput([]byte(`{"posts":[{}]}`))
				require.NoError(t, err)
				require.Empty(t, parsed.Posts)
			},
		},
		"flatten skips empty ids and calls callback": {
			input: func() {
				raw := []byte(`{"posts":[{"post":{"post_id":"","message":"skip"}},{"post":{"post_id":"p1","message":"hello https://example.com","attachments":[{"type":"link"},{"type":"photo","url":"https://img","media_url":"https://cdn","width":0,"height":2}]},"comments":{"comments":[{"id":"","message":"skip"},{"id":"c1","message":"comment","created_time":0,"replies":[{"id":"r1"}]}]}}]}`)
				called := 0
				records, err := uc.flattenFacebookFullFlow(raw, input, func(uap.UAPRecord) { called++ })
				require.NoError(t, err)
				require.Len(t, records, 2)
				require.Equal(t, 2, called)
			},
		},
		"direct empty mapping helpers": {
			input: func() {
				_, rootID := uc.mapFacebookPost(uap.FacebookPostBundleInput{}, input)
				require.Empty(t, rootID)
				require.Empty(t, uc.mapFacebookComment(uap.FacebookCommentInput{}, input, "root").Identity.OriginID)
				require.Nil(t, uc.mapFacebookAttachments(nil))
				require.Nil(t, uc.mapFacebookAttachments([]uap.FacebookAttachmentInput{{Type: "link"}}))
				require.Empty(t, uc.normalizeFacebookUnixTime(0))
			},
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			tc.input()
		})
	}
}

func TestTikTokParserEdges(t *testing.T) {
	uc := &implUseCase{}
	input := uap.ParseAndStoreRawBatchInput{TaskID: "task-1", ProjectID: "project-1", CompletionTime: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}

	tcs := map[string]struct {
		input func()
		err   error
	}{
		"flatten invalid json": {
			input: func() {
				_, err := uc.flattenTikTokFullFlow([]byte(`{`), input, nil)
				require.ErrorIs(t, err, uap.ErrParseRawPayload)
			},
		},
		"flatten skips empty ids and calls callback": {
			input: func() {
				raw := []byte(`{"result":{"posts":[{"post":{"video_id":""}},{"post":{"video_id":"v1","description":"post"},"comments":{"comments":[{"comment_id":"","reply_comments":[{"reply_id":"skip"}]},{"comment_id":"c1","content":"comment","sort_extra_score":{"reply_score":1},"reply_comments":[{"reply_id":"","content":"skip"},{"reply_id":"r1","content":"reply"}]}]}}]}}`)
				called := 0
				records, err := uc.flattenTikTokFullFlow(raw, input, func(uap.UAPRecord) { called++ })
				require.NoError(t, err)
				require.Len(t, records, 3)
				require.Equal(t, 3, called)
			},
		},
		"direct empty mapping helpers": {
			input: func() {
				_, rootID := uc.mapTikTokPost(uap.TikTokPostBundleInput{}, input)
				require.Empty(t, rootID)
				_, commentID := uc.mapTikTokComment(uap.TikTokCommentInput{}, input, "root")
				require.Empty(t, commentID)
				require.Empty(t, uc.mapTikTokReply(uap.TikTokReplyInput{}, input, "root", "comment").Identity.OriginID)
			},
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			tc.input()
		})
	}
}

func TestYouTubeParserEdges(t *testing.T) {
	uc := &implUseCase{}
	input := uap.ParseAndStoreRawBatchInput{TaskID: "task-1", ProjectID: "project-1", CompletionTime: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}

	tcs := map[string]struct {
		input func()
		err   error
	}{
		"flatten invalid json": {
			input: func() {
				_, err := uc.flattenYouTubeFullFlow([]byte(`{`), input, nil)
				require.ErrorIs(t, err, uap.ErrParseRawPayload)
			},
		},
		"parse skips empty video map": {
			input: func() {
				parsed, err := uc.parseYouTubeFullFlowInput([]byte(`{"videos":[{}]}`))
				require.NoError(t, err)
				require.Empty(t, parsed.Videos)
			},
		},
		"flatten skips empty ids and calls callback": {
			input: func() {
				raw := []byte(`{"videos":[{"video":{"video_id":""}},{"video":{"video_id":"v1","title":"title","channel_id":"ch1"},"comments":{"comments":[{"comment_id":"","content":"skip"},{"comment_id":"c1","content":"comment https://example.com","author_channel_id":"ch2"}]}}]}`)
				called := 0
				records, err := uc.flattenYouTubeFullFlow(raw, input, func(uap.UAPRecord) { called++ })
				require.NoError(t, err)
				require.Len(t, records, 2)
				require.Equal(t, 2, called)
			},
		},
		"direct empty mapping helpers": {
			input: func() {
				_, rootID := uc.mapYouTubePost(uap.YouTubeVideoBundleInput{}, input)
				require.Empty(t, rootID)
				require.Empty(t, uc.mapYouTubeComment(uap.YouTubeCommentInput{}, input, "root").Identity.OriginID)
			},
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			tc.input()
		})
	}
}
