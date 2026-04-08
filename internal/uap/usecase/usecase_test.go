package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"ingest-srv/internal/uap"
)

func TestChunkRecords(t *testing.T) {
	records := make([]uap.UAPRecord, 21)
	for i := range records {
		records[i] = uap.UAPRecord{
			Identity: uap.UAPIdentity{UAPID: "id"},
		}
	}

	uc := &implUseCase{}
	chunks := uc.chunkRecords(records)
	if len(chunks) != 2 {
		t.Fatalf("expected 2 chunks, got %d", len(chunks))
	}
	if len(chunks[0]) != 20 {
		t.Fatalf("expected first chunk size 20, got %d", len(chunks[0]))
	}
	if len(chunks[1]) != 1 {
		t.Fatalf("expected second chunk size 1, got %d", len(chunks[1]))
	}
}

func TestMergeRawMetadata(t *testing.T) {
	existing := json.RawMessage(`{"crawler_version":"1.0.0"}`)
	uc := &implUseCase{}
	metadata, err := uc.mergeRawMetadata(existing, []uap.ArtifactPart{
		{
			PartNo:        1,
			StorageBucket: "ingest-data",
			StoragePath:   "uap-batches/project-1/source-1/batch-1/part-00001.jsonl",
			RecordCount:   20,
		},
	}, 20, &uap.KafkaPublishStats{
		Topic:          "smap.collector.output",
		AttemptedCount: 20,
		SuccessCount:   19,
		FailedCount:    1,
		LastError:      "boom",
	})
	if err != nil {
		t.Fatalf("mergeRawMetadata() error = %v", err)
	}

	var root map[string]interface{}
	if err := json.Unmarshal(metadata, &root); err != nil {
		t.Fatalf("metadata unmarshal error = %v", err)
	}

	if root["crawler_version"] != "1.0.0" {
		t.Fatalf("expected crawler_version to be preserved")
	}
	artifacts, ok := root[uap.ArtifactsMetadataKey].(map[string]interface{})
	if !ok {
		t.Fatalf("expected uap_artifacts map")
	}
	if int(artifacts["chunk_size"].(float64)) != uap.ChunkSize {
		t.Fatalf("unexpected chunk size: %#v", artifacts["chunk_size"])
	}
	if int(artifacts["total_parts"].(float64)) != 1 {
		t.Fatalf("unexpected total_parts: %#v", artifacts["total_parts"])
	}

	publish, ok := root[uap.KafkaPublishKey].(map[string]interface{})
	if !ok {
		t.Fatalf("expected kafka_publish map")
	}
	if publish["topic"] != "smap.collector.output" {
		t.Fatalf("unexpected publish topic: %#v", publish["topic"])
	}
	if int(publish["failed_count"].(float64)) != 1 {
		t.Fatalf("unexpected failed_count: %#v", publish["failed_count"])
	}
}

func TestPublishPayloadUsesCurrentUAPRecord(t *testing.T) {
	record := uap.UAPRecord{
		Identity: uap.UAPIdentity{
			UAPID:    "tt_p_7601",
			OriginID: "7601",
			UAPType:  uap.UAPTypePost,
			Platform: uap.PlatformTikTok,
			URL:      "https://www.tiktok.com/@demo/video/7601",
			TaskID:   "task-1",
		},
		Hierarchy: uap.UAPHierarchy{
			RootID: "tt_p_7601",
			Depth:  0,
		},
		Content: uap.UAPContent{Text: "video text", Title: "title", Subtitle: "subtitle", Language: "vi", Keywords: []string{"vinfast"}},
		Author: uap.UAPAuthor{
			ID:         "author-1",
			Username:   "demo_author",
			Nickname:   "Demo Author",
			ProfileURL: "",
		},
		Engagement: uap.UAPEngagement{
			Likes:         ptrInt(10),
			CommentsCount: ptrInt(5),
			Shares:        ptrInt(2),
			Views:         ptrInt(100),
			Saves:         ptrInt(3),
		},
		Media: []uap.UAPMedia{
			{
				Type:        "video",
				DownloadURL: "https://example.com/video.mp4",
			},
		},
		Temporal: uap.UAPTemporal{
			PostedAt: "2026-03-08T00:00:00Z",
		},
		CrawlKeyword: "bia heineken",
		PlatformMeta: map[string]interface{}{"tiktok": map[string]interface{}{"is_shop_video": false}},
	}

	input := uap.ParseAndStoreRawBatchInput{
		RawBatchID: "raw-batch-1",
	}

	spy := &spyPublisher{}
	stats := &uap.KafkaPublishStats{Topic: "smap.collector.output"}
	uc := &implUseCase{publisher: spy, publishTopic: "smap.collector.output"}

	uc.publishRecord(context.Background(), record, input, stats)

	if stats.AttemptedCount != 0 || stats.SuccessCount != 0 || stats.FailedCount != 0 {
		t.Fatalf("unexpected publish stats: %#v", stats)
	}
	if spy.lastInput.Record.Identity.UAPID != "" {
		t.Fatalf("expected no publish call while publishRecord is disabled, got %#v", spy.lastInput.Record)
	}
}

func TestMarshalUAPRecordUsesVNextKeysOnly(t *testing.T) {
	record := uap.UAPRecord{
		Identity: uap.UAPIdentity{
			UAPID:    "tt_p_7601",
			OriginID: "7601",
			UAPType:  uap.UAPTypePost,
			Platform: uap.PlatformTikTok,
		},
		Hierarchy: uap.UAPHierarchy{
			RootID: "tt_p_7601",
			Depth:  0,
		},
		Content: uap.UAPContent{
			Text:     "video text",
			Title:    "title",
			Subtitle: "subtitle",
			Keywords: []string{"vinfast"},
			Links:    []string{"https://example.com"},
		},
		Author: uap.UAPAuthor{
			ID:         "author-1",
			Username:   "demo_author",
			Nickname:   "Demo Author",
			ProfileURL: "https://example.com/profile",
		},
		Engagement: uap.UAPEngagement{
			Saves:      ptrInt(3),
			ReplyCount: ptrInt(2),
		},
		Temporal: uap.UAPTemporal{
			PostedAt:  "2026-03-08T00:00:00Z",
			UpdatedAt: "",
		},
		CrawlKeyword: "bia tiger",
		PlatformMeta: map[string]interface{}{"tiktok": map[string]interface{}{"sort_score": 0.75}},
	}

	uc := &implUseCase{}
	body, err := uc.marshalUAPRecord(record)
	if err != nil {
		t.Fatalf("marshalUAPRecord() error = %v", err)
	}

	var root map[string]interface{}
	if err := json.Unmarshal(body, &root); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	content := root["content"].(map[string]interface{})
	if _, ok := content["title"]; !ok {
		t.Fatalf("expected vNext title field")
	}
	if _, ok := content["subtitle"]; !ok {
		t.Fatalf("expected vNext subtitle field")
	}
	if _, ok := content["keywords"]; !ok {
		t.Fatalf("expected vNext keywords field")
	}
	if _, ok := content["links"]; !ok {
		t.Fatalf("expected vNext links field")
	}

	engagement := root["engagement"].(map[string]interface{})
	if _, ok := engagement["saves"]; !ok {
		t.Fatalf("expected vNext saves field")
	}

	if _, ok := root["platform_meta"]; !ok {
		t.Fatalf("expected platform_meta field")
	}
	if root["crawl_keyword"] != "bia tiger" {
		t.Fatalf("expected crawl_keyword field, got %#v", root["crawl_keyword"])
	}

	for _, legacyKey := range []string{"tiktok_keywords", "summary_title", "subtitle_url", "external_links", "music_title", "music_url", "is_shop_video"} {
		if _, ok := content[legacyKey]; ok {
			t.Fatalf("legacy key %q should be absent", legacyKey)
		}
	}
	if _, ok := engagement["bookmarks"]; ok {
		t.Fatalf("legacy key bookmarks should be absent")
	}
	if _, ok := engagement["sort_score"]; ok {
		t.Fatalf("legacy key sort_score should be absent")
	}
}

func TestNormalizeWEBVTT(t *testing.T) {
	uc := &implUseCase{}
	body := []byte("WEBVTT\n\n1\n00:00:00.000 --> 00:00:01.000\nXin chao\n\n2\n00:00:01.000 --> 00:00:02.000\nMoi nguoi\n")

	got := uc.normalizeWEBVTT(body)
	want := "Xin chao Moi nguoi"
	if got != want {
		t.Fatalf("normalizeWEBVTT() = %q, want %q", got, want)
	}
}

func TestDownloadSubtitleTextAcceptsWrongContentType(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "video/mp4")
		_, _ = w.Write([]byte("WEBVTT\n\n00:00:00.000 --> 00:00:01.000\nHello world\n"))
	}))
	defer server.Close()

	uc := &implUseCase{subtitleHTTPClient: server.Client()}
	got, err := uc.downloadSubtitleText(server.URL)
	if err != nil {
		t.Fatalf("downloadSubtitleText() error = %v", err)
	}
	if got != "Hello world" {
		t.Fatalf("downloadSubtitleText() = %q, want %q", got, "Hello world")
	}
}

func TestDownloadSubtitleTextReturnsEmptyFor404OrOversize(t *testing.T) {
	oversized := strings.Repeat("a", maxSubtitleBodyBytes+2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/404":
			http.NotFound(w, r)
		case "/oversize":
			_, _ = w.Write([]byte(oversized))
		default:
			http.Error(w, "unexpected", http.StatusInternalServerError)
		}
	}))
	defer server.Close()

	uc := &implUseCase{subtitleHTTPClient: server.Client()}
	for _, url := range []string{server.URL + "/404", server.URL + "/oversize"} {
		got, err := uc.downloadSubtitleText(url)
		if err == nil {
			t.Fatalf("downloadSubtitleText(%q) expected error", url)
		}
		if got != "" {
			t.Fatalf("downloadSubtitleText(%q) = %q, want empty", url, got)
		}
	}
}

func TestResolveSubtitleTextFallsBackAndNeverFails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/primary":
			http.NotFound(w, r)
		case "/fallback":
			_, _ = w.Write([]byte("WEBVTT\n\n00:00:00.000 --> 00:00:01.000\nFallback subtitle\n"))
		default:
			http.Error(w, "unexpected", http.StatusInternalServerError)
		}
	}))
	defer server.Close()

	uc := &implUseCase{subtitleHTTPClient: server.Client()}
	got := uc.resolveSubtitleText(server.URL+"/primary", server.URL+"/fallback")
	if got != "Fallback subtitle" {
		t.Fatalf("resolveSubtitleText() = %q, want %q", got, "Fallback subtitle")
	}

	empty := uc.resolveSubtitleText(fmt.Sprintf("%s/%s", server.URL, "404"))
	if empty != "" {
		t.Fatalf("resolveSubtitleText() = %q, want empty", empty)
	}
}

func TestResolveTikTokSubtitleTextUsesTikTokFallbackOrder(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/subtitle-primary":
			http.NotFound(w, r)
		case "/subtitle-fallback":
			_, _ = w.Write([]byte("WEBVTT\n\n00:00:00.000 --> 00:00:01.000\nTikTok subtitle\n"))
		default:
			http.Error(w, "unexpected", http.StatusInternalServerError)
		}
	}))
	defer server.Close()

	uc := &implUseCase{subtitleHTTPClient: server.Client()}
	got := uc.resolveTikTokSubtitleText(uap.TikTokDetailInput{
		SubtitleURL: server.URL + "/subtitle-primary",
		Downloads: uap.TikTokDetailAssetsInput{
			Subtitle: server.URL + "/subtitle-fallback",
		},
	})
	if got != "TikTok subtitle" {
		t.Fatalf("resolveTikTokSubtitleText() = %q, want %q", got, "TikTok subtitle")
	}
}

type spyPublisher struct {
	lastInput uap.PublishUAPInput
}

func (s *spyPublisher) Publish(_ context.Context, input uap.PublishUAPInput) error {
	s.lastInput = input
	return nil
}

func (s *spyPublisher) Topic() string {
	return "smap.collector.output"
}

func (s *spyPublisher) Close() error {
	return nil
}

func ptrInt(v int) *int {
	return &v
}

func TestFlattenTikTokFullFlow(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("WEBVTT\n\n00:00:00.000 --> 00:00:01.000\nsubtitle text\n"))
	}))
	defer server.Close()

	raw := []byte(`{
		"result": {
			"posts": [
				{
					"post": {
						"video_id": "7601",
						"url": "https://www.tiktok.com/@demo/video/7601",
						"description": "post body",
						"author": {
							"uid": "author-1",
							"username": "demo_author",
							"nickname": "Demo Author",
							"avatar": "https://example.com/a.jpg"
						},
						"likes_count": 10,
						"comments_count": 1,
						"shares_count": 2,
						"views_count": 99,
						"hashtags": ["vf8"],
						"posted_at": "2026-03-08T00:00:00Z",
						"is_shop_video": false
					},
					"detail": {
						"video_id": "7601",
						"url": "https://www.tiktok.com/@demo/video/7601",
						"description": "post body detail",
						"author": {
							"uid": "author-1",
							"username": "demo_author",
							"nickname": "Demo Author",
							"avatar": "https://example.com/a.jpg"
						},
						"likes_count": 10,
						"comments_count": 1,
						"shares_count": 2,
						"views_count": 99,
						"bookmarks_count": 5,
						"hashtags": ["vf8"],
						"music_title": "demo sound",
						"music_url": "https://example.com/music.mp3",
						"duration": 84,
						"posted_at": "2026-03-08T00:00:00Z",
						"is_shop_video": false,
						"summary": {
							"title": "Demo Summary",
							"keywords": ["vinfast vf8", "review"],
							"language": "vi"
						},
						"play_url": "https://example.com/play.mp4",
						"download_url": "https://example.com/video.mp4",
						"cover_url": "https://example.com/cover.jpg",
						"subtitle_url": "` + server.URL + `/subtitle.vtt",
						"downloads": {
							"music": "https://example.com/music-download.mp3",
							"cover": "https://example.com/cover-download.jpg",
							"subtitle": "` + server.URL + `/subtitle-download.vtt",
							"video": "https://example.com/video-download.mp4"
						}
					},
					"comments": {
						"comments": [
							{
								"comment_id": "c-1",
								"content": "comment body",
								"author": {
									"uid": "commenter-1",
									"username": "commenter",
									"nickname": "Commenter",
									"avatar": "https://example.com/c.jpg"
								},
								"likes_count": 7,
								"reply_count": 1,
								"sort_extra_score": {
									"reply_score": 0.25,
									"show_more_score": 0.75
								},
								"commented_at": "2026-03-08T01:00:00Z",
								"reply_comments": [
									{
										"reply_id": "r-1",
										"content": "reply body",
										"author": {
											"uid": "replier-1",
											"username": "replier",
											"nickname": "Replier",
											"avatar": "https://example.com/r.jpg"
										},
										"likes_count": 3,
										"replied_at": "2026-03-08T01:10:00Z"
									}
								]
							}
						]
					}
				}
			]
		}
	}`)

	input := uap.ParseAndStoreRawBatchInput{
		ProjectID:      "project-1",
		TaskID:         "task-1",
		Platform:       "tiktok",
		Action:         "full_flow",
		RequestPayload: json.RawMessage(`{"params":{"keyword":"bia heineken"}}`),
		CompletionTime: time.Date(2026, time.March, 8, 4, 0, 0, 0, time.UTC),
	}

	uc := &implUseCase{subtitleHTTPClient: server.Client()}
	records, err := uc.flattenTikTokFullFlow(raw, input, nil)
	if err != nil {
		t.Fatalf("flattenTikTokFullFlow() error = %v", err)
	}
	if len(records) != 3 {
		t.Fatalf("expected 3 records, got %d", len(records))
	}

	post := records[0]
	if post.Identity.UAPID != "tt_p_7601" {
		t.Fatalf("unexpected post uap_id: %s", post.Identity.UAPID)
	}
	if post.Hierarchy.Depth != 0 {
		t.Fatalf("unexpected post depth: %d", post.Hierarchy.Depth)
	}
	if post.Identity.ProjectID != "project-1" {
		t.Fatalf("unexpected post project_id: %s", post.Identity.ProjectID)
	}
	if post.Content.Title != "Demo Summary" {
		t.Fatalf("unexpected post title: %s", post.Content.Title)
	}
	if post.Content.Subtitle != "subtitle text" {
		t.Fatalf("unexpected post subtitle: %s", post.Content.Subtitle)
	}
	if len(post.Content.Keywords) != 2 {
		t.Fatalf("unexpected post keywords: %#v", post.Content.Keywords)
	}
	if post.Engagement.Saves == nil || *post.Engagement.Saves != 5 {
		t.Fatalf("unexpected post saves: %#v", post.Engagement.Saves)
	}
	if post.CrawlKeyword != "bia heineken" {
		t.Fatalf("unexpected post crawl_keyword: %s", post.CrawlKeyword)
	}
	tiktokMeta, ok := post.PlatformMeta["tiktok"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected post platform_meta.tiktok")
	}
	if tiktokMeta["music_title"] != "demo sound" {
		t.Fatalf("unexpected post platform_meta music_title: %#v", tiktokMeta["music_title"])
	}

	comment := records[1]
	if comment.Identity.UAPID != "tt_c_c-1" {
		t.Fatalf("unexpected comment uap_id: %s", comment.Identity.UAPID)
	}
	if comment.Hierarchy.ParentID == nil || *comment.Hierarchy.ParentID != "tt_p_7601" {
		t.Fatalf("unexpected comment parent_id: %#v", comment.Hierarchy.ParentID)
	}
	if comment.Hierarchy.Depth != 1 {
		t.Fatalf("unexpected comment depth: %d", comment.Hierarchy.Depth)
	}
	if comment.Engagement.ReplyCount == nil || *comment.Engagement.ReplyCount != 1 {
		t.Fatalf("unexpected comment reply_count: %#v", comment.Engagement.ReplyCount)
	}
	commentMeta, ok := comment.PlatformMeta["tiktok"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected comment platform_meta.tiktok")
	}
	if commentMeta["sort_score"] != 0.75 {
		t.Fatalf("unexpected comment sort_score: %#v", commentMeta["sort_score"])
	}
	if comment.CrawlKeyword != "bia heineken" {
		t.Fatalf("unexpected comment crawl_keyword: %s", comment.CrawlKeyword)
	}

	reply := records[2]
	if reply.Identity.UAPID != "tt_r_r-1" {
		t.Fatalf("unexpected reply uap_id: %s", reply.Identity.UAPID)
	}
	if reply.Hierarchy.ParentID == nil || *reply.Hierarchy.ParentID != "tt_c_c-1" {
		t.Fatalf("unexpected reply parent_id: %#v", reply.Hierarchy.ParentID)
	}
	if reply.Hierarchy.RootID != "tt_p_7601" {
		t.Fatalf("unexpected reply root_id: %s", reply.Hierarchy.RootID)
	}
	if reply.Hierarchy.Depth != 2 {
		t.Fatalf("unexpected reply depth: %d", reply.Hierarchy.Depth)
	}
	if reply.CrawlKeyword != "bia heineken" {
		t.Fatalf("unexpected reply crawl_keyword: %s", reply.CrawlKeyword)
	}
}

func TestSupportsParse(t *testing.T) {
	uc := &implUseCase{}

	if !uc.SupportsParse("tiktok", "full_flow") {
		t.Fatalf("expected tiktok/full_flow to be supported")
	}
	if !uc.SupportsParse(" TiKtOk ", " FULL_FLOW ") {
		t.Fatalf("expected SupportsParse to be case-insensitive and trim spaces")
	}
	if !uc.SupportsParse("youtube", "full_flow") {
		t.Fatalf("expected youtube/full_flow to be supported")
	}
	if !uc.SupportsParse("facebook", "full_flow") {
		t.Fatalf("expected facebook/full_flow to be supported")
	}
	if uc.SupportsParse("tiktok", "comments") {
		t.Fatalf("expected tiktok/comments to be unsupported")
	}
}

func TestResolveParser(t *testing.T) {
	uc := &implUseCase{}

	parser, ok := uc.resolveParser("tiktok", "full_flow")
	if !ok {
		t.Fatalf("expected parser lookup to succeed for tiktok/full_flow")
	}
	if parser == nil {
		t.Fatalf("expected non-nil parser for tiktok/full_flow")
	}

	parser, ok = uc.resolveParser("youtube", "full_flow")
	if !ok {
		t.Fatalf("expected parser lookup to succeed for youtube/full_flow")
	}
	if parser == nil {
		t.Fatalf("expected non-nil parser for youtube/full_flow")
	}

	parser, ok = uc.resolveParser("facebook", "full_flow")
	if !ok {
		t.Fatalf("expected parser lookup to succeed for facebook/full_flow")
	}
	if parser == nil {
		t.Fatalf("expected non-nil parser for facebook/full_flow")
	}
}

func TestFlattenYouTubeFullFlow(t *testing.T) {
	raw := []byte(`{
		"result": {
			"videos": [
				{
					"video": {
						"video_id": "yt-1",
						"title": "Video title",
						"channel_name": "Channel Name",
						"channel_id": "channel-1",
						"views_count": 1234,
						"views_text": "1.234 views",
						"duration_text": "7:11",
						"published_time_text": "10 hours ago",
						"thumbnail_url": "https://example.com/thumb.jpg",
						"description_snippet": "Video snippet https://example.com/snippet",
						"url": "https://www.youtube.com/watch?v=yt-1"
					},
					"detail": {
						"video_id": "yt-1",
						"title": "Detail title",
						"description": "Detail description https://example.com/detail",
						"keywords": ["iphone 16", "review"],
						"width": 1280,
						"height": 720,
						"author_name": "Detail Author",
						"author_url": "http://www.youtube.com/@detail_author",
						"likes_count": 42,
						"views_count": 2222,
						"date_published": "2026-03-23T01:04:03-07:00",
						"upload_date": "2026-03-23T01:04:03-07:00"
					},
					"comments": {
						"video_id": "yt-1",
						"total": 1,
						"comments": [
							{
								"comment_id": "c-1",
								"video_id": "yt-1",
								"author_name": "@commenter",
								"author_channel_id": "commenter-1",
								"author_thumbnail_url": "https://example.com/commenter.jpg",
								"content": "Comment with link https://example.com/comment",
								"likes_count": 3,
								"reply_count": 2,
								"published_time_text": "6 hours ago"
							}
						]
					},
					"transcript": {
						"full_text": "Transcript text https://example.com/transcript",
						"segments": [
							{
								"start_ms": 0,
								"end_ms": 1000,
								"text": "ignored fallback",
								"start_time_text": "0:00"
							}
						]
					}
				}
			]
		}
	}`)

	input := uap.ParseAndStoreRawBatchInput{
		ProjectID:      "project-1",
		TaskID:         "task-1",
		Platform:       "youtube",
		Action:         "full_flow",
		RequestPayload: json.RawMessage(`{"params":{"keyword":"bia sapporo"}}`),
		CompletionTime: time.Date(2026, time.March, 24, 10, 0, 0, 0, time.UTC),
	}

	uc := &implUseCase{}
	records, err := uc.flattenYouTubeFullFlow(raw, input, nil)
	if err != nil {
		t.Fatalf("flattenYouTubeFullFlow() error = %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(records))
	}

	post := records[0]
	if post.Identity.UAPID != "yt_p_yt-1" {
		t.Fatalf("unexpected post uap_id: %s", post.Identity.UAPID)
	}
	if post.Content.Text != "Detail description https://example.com/detail" {
		t.Fatalf("unexpected post text: %s", post.Content.Text)
	}
	if post.Content.Title != "Detail title" {
		t.Fatalf("unexpected post title: %s", post.Content.Title)
	}
	if post.Content.Subtitle != "Transcript text https://example.com/transcript" {
		t.Fatalf("unexpected post subtitle: %s", post.Content.Subtitle)
	}
	if len(post.Content.Links) != 3 {
		t.Fatalf("unexpected post links: %#v", post.Content.Links)
	}
	if post.Author.Username != "@detail_author" {
		t.Fatalf("unexpected post username: %s", post.Author.Username)
	}
	if post.Author.ProfileURL != "http://www.youtube.com/@detail_author" {
		t.Fatalf("unexpected post profile_url: %s", post.Author.ProfileURL)
	}
	if post.Engagement.CommentsCount == nil || *post.Engagement.CommentsCount != 1 {
		t.Fatalf("unexpected post comments_count: %#v", post.Engagement.CommentsCount)
	}
	if len(post.Media) != 1 {
		t.Fatalf("expected 1 media item, got %d", len(post.Media))
	}
	if post.Media[0].Duration == nil || *post.Media[0].Duration != 431 {
		t.Fatalf("unexpected post duration: %#v", post.Media[0].Duration)
	}
	if post.Media[0].Width == nil || *post.Media[0].Width != 1280 {
		t.Fatalf("unexpected post width: %#v", post.Media[0].Width)
	}
	if post.Media[0].Height == nil || *post.Media[0].Height != 720 {
		t.Fatalf("unexpected post height: %#v", post.Media[0].Height)
	}
	if post.CrawlKeyword != "bia sapporo" {
		t.Fatalf("unexpected post crawl_keyword: %s", post.CrawlKeyword)
	}

	comment := records[1]
	if comment.Identity.UAPID != "yt_c_c-1" {
		t.Fatalf("unexpected comment uap_id: %s", comment.Identity.UAPID)
	}
	if comment.Hierarchy.ParentID == nil || *comment.Hierarchy.ParentID != "yt_p_yt-1" {
		t.Fatalf("unexpected comment parent_id: %#v", comment.Hierarchy.ParentID)
	}
	if comment.Engagement.ReplyCount == nil || *comment.Engagement.ReplyCount != 2 {
		t.Fatalf("unexpected comment reply_count: %#v", comment.Engagement.ReplyCount)
	}
	if comment.Content.Subtitle != "" {
		t.Fatalf("expected empty comment subtitle, got %q", comment.Content.Subtitle)
	}
	if comment.CrawlKeyword != "bia sapporo" {
		t.Fatalf("unexpected comment crawl_keyword: %s", comment.CrawlKeyword)
	}
}

func TestFlattenYouTubeFullFlowFallbackTranscriptSegments(t *testing.T) {
	raw := []byte(`{
		"videos": [
			{
				"video": {
					"video_id": "yt-2",
					"title": "Video title",
					"channel_name": "Channel Name",
					"channel_id": "channel-2",
					"duration_text": "1:02",
					"url": "https://www.youtube.com/watch?v=yt-2"
				},
				"detail": {
					"description": "desc"
				},
				"comments": {
					"total": 0,
					"comments": []
				},
				"transcript": {
					"full_text": "",
					"segments": [
						{"text": "first"},
						{"text": "second"}
					]
				}
			}
		]
	}`)

	input := uap.ParseAndStoreRawBatchInput{
		ProjectID:      "project-1",
		TaskID:         "task-1",
		Platform:       "youtube",
		Action:         "full_flow",
		RequestPayload: json.RawMessage(`{"params":{"keyword":"bia larue"}}`),
		CompletionTime: time.Date(2026, time.March, 24, 10, 0, 0, 0, time.UTC),
	}

	uc := &implUseCase{}
	records, err := uc.flattenYouTubeFullFlow(raw, input, nil)
	if err != nil {
		t.Fatalf("flattenYouTubeFullFlow() error = %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	if records[0].Content.Subtitle != "first second" {
		t.Fatalf("unexpected subtitle fallback: %q", records[0].Content.Subtitle)
	}
	if records[0].Media[0].Duration == nil || *records[0].Media[0].Duration != 62 {
		t.Fatalf("unexpected duration fallback: %#v", records[0].Media[0].Duration)
	}
	if records[0].CrawlKeyword != "bia larue" {
		t.Fatalf("unexpected crawl_keyword fallback case: %s", records[0].CrawlKeyword)
	}
}

func TestFlattenYouTubeFullFlowDoesNotEmitReplyRecords(t *testing.T) {
	raw := []byte(`{
		"videos": [
			{
				"video": {
					"video_id": "yt-3",
					"title": "Video title",
					"url": "https://www.youtube.com/watch?v=yt-3"
				},
				"comments": {
					"total": 2,
					"comments": [
						{"comment_id": "c-1", "reply_count": 5},
						{"comment_id": "c-2", "reply_count": 1}
					]
				}
			}
		]
	}`)

	input := uap.ParseAndStoreRawBatchInput{
		ProjectID:      "project-1",
		TaskID:         "task-1",
		Platform:       "youtube",
		Action:         "full_flow",
		RequestPayload: json.RawMessage(`{"params":{"keyword":"bia hoegaarden"}}`),
		CompletionTime: time.Date(2026, time.March, 24, 10, 0, 0, 0, time.UTC),
	}

	uc := &implUseCase{}
	records, err := uc.flattenYouTubeFullFlow(raw, input, nil)
	if err != nil {
		t.Fatalf("flattenYouTubeFullFlow() error = %v", err)
	}
	if len(records) != 3 {
		t.Fatalf("expected 3 records, got %d", len(records))
	}
	for _, record := range records {
		if record.Identity.UAPType == uap.UAPTypeReply {
			t.Fatalf("unexpected REPLY record: %#v", record)
		}
		if record.CrawlKeyword != "bia hoegaarden" {
			t.Fatalf("unexpected crawl_keyword on youtube reply test: %s", record.CrawlKeyword)
		}
	}
}

func TestFlattenYouTubeFullFlowToleratesMissingBlocks(t *testing.T) {
	raw := []byte(`{
		"result": {
			"videos": [
				{
					"video": {
						"video_id": "yt-4",
						"title": "Fallback title",
						"description_snippet": "Fallback snippet",
						"channel_id": "channel-4",
						"channel_name": "Fallback Channel",
						"url": "https://www.youtube.com/watch?v=yt-4"
					},
					"detail": {"error": "boom"},
					"comments": {"error": "boom"},
					"transcript": null
				}
			]
		}
	}`)

	input := uap.ParseAndStoreRawBatchInput{
		ProjectID:      "project-1",
		TaskID:         "task-1",
		Platform:       "youtube",
		Action:         "full_flow",
		RequestPayload: json.RawMessage(`{"params":{"keyword":"bia corona"}}`),
		CompletionTime: time.Date(2026, time.March, 24, 10, 0, 0, 0, time.UTC),
	}

	uc := &implUseCase{}
	records, err := uc.flattenYouTubeFullFlow(raw, input, nil)
	if err != nil {
		t.Fatalf("flattenYouTubeFullFlow() error = %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	post := records[0]
	if post.Content.Text != "Fallback snippet" {
		t.Fatalf("unexpected fallback text: %q", post.Content.Text)
	}
	if post.Content.Title != "Fallback title" {
		t.Fatalf("unexpected fallback title: %q", post.Content.Title)
	}
	if post.Author.ProfileURL != "https://www.youtube.com/channel/channel-4" {
		t.Fatalf("unexpected fallback profile_url: %q", post.Author.ProfileURL)
	}
	if post.Content.Subtitle != "" {
		t.Fatalf("expected empty subtitle, got %q", post.Content.Subtitle)
	}
	if post.CrawlKeyword != "bia corona" {
		t.Fatalf("unexpected crawl_keyword missing blocks case: %s", post.CrawlKeyword)
	}
}

func TestFlattenFacebookFullFlowPhotoAttachment(t *testing.T) {
	raw := []byte(`{
		"result": {
			"posts": [
				{
					"post": {
						"post_id": "fb-1",
						"message": "Photo post https://example.com/post",
						"url": "https://www.facebook.com/posts/fb-1",
						"author": {
							"id": "author-1",
							"name": "Facebook Page",
							"url": "https://www.facebook.com/page",
							"avatar_url": "https://example.com/avatar.jpg"
						},
						"created_time": 1711267200,
						"reaction_count": 10,
						"comment_count": 1,
						"share_count": 2,
						"attachments": [
							{
								"type": "Photo",
								"url": "https://www.facebook.com/photo/fb-1",
								"media_url": "https://example.com/photo.jpg",
								"width": 1200,
								"height": 630,
								"accessibility_caption": "ignored"
							}
						]
					},
					"comments": {
						"post_id": "fb-1",
						"total_count": 1,
						"comments": [
							{
								"id": "fc-1",
								"message": "Comment body https://example.com/comment",
								"author": {
									"id": "commenter-1",
									"name": "Commenter",
									"profile_url": "https://www.facebook.com/commenter",
									"avatar_url": "https://example.com/commenter.jpg"
								},
								"created_time": 1711267800,
								"reaction_count": 3,
								"reply_count": 0,
								"replies": null
							}
						]
					}
				}
			]
		}
	}`)

	input := uap.ParseAndStoreRawBatchInput{
		ProjectID:      "project-1",
		TaskID:         "task-1",
		Platform:       "facebook",
		Action:         "full_flow",
		RequestPayload: json.RawMessage(`{"params":{"keyword":"bia tiger"}}`),
		CompletionTime: time.Date(2026, time.March, 24, 12, 0, 0, 0, time.UTC),
	}

	uc := &implUseCase{}
	records, err := uc.flattenFacebookFullFlow(raw, input, nil)
	if err != nil {
		t.Fatalf("flattenFacebookFullFlow() error = %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(records))
	}

	post := records[0]
	if post.Identity.UAPID != "fb_p_fb-1" {
		t.Fatalf("unexpected post uap_id: %s", post.Identity.UAPID)
	}
	if post.Author.ProfileURL != "https://www.facebook.com/page" {
		t.Fatalf("unexpected post profile_url: %s", post.Author.ProfileURL)
	}
	if post.Content.Subtitle != "" {
		t.Fatalf("expected empty post subtitle, got %q", post.Content.Subtitle)
	}
	if len(post.Media) != 1 {
		t.Fatalf("expected 1 media item, got %d", len(post.Media))
	}
	if post.Media[0].Type != "image" {
		t.Fatalf("unexpected media type: %s", post.Media[0].Type)
	}
	if post.Media[0].DownloadURL != "https://example.com/photo.jpg" {
		t.Fatalf("unexpected media download_url: %s", post.Media[0].DownloadURL)
	}
	if post.Media[0].Width == nil || *post.Media[0].Width != 1200 {
		t.Fatalf("unexpected media width: %#v", post.Media[0].Width)
	}
	if post.Temporal.PostedAt != "2024-03-24T08:00:00Z" {
		t.Fatalf("unexpected post posted_at: %s", post.Temporal.PostedAt)
	}
	if post.CrawlKeyword != "bia tiger" {
		t.Fatalf("unexpected post crawl_keyword: %s", post.CrawlKeyword)
	}

	comment := records[1]
	if comment.Identity.UAPID != "fb_c_fc-1" {
		t.Fatalf("unexpected comment uap_id: %s", comment.Identity.UAPID)
	}
	if comment.Author.ProfileURL != "https://www.facebook.com/commenter" {
		t.Fatalf("unexpected comment profile_url: %s", comment.Author.ProfileURL)
	}
	if len(comment.Content.Links) != 1 || comment.Content.Links[0] != "https://example.com/comment" {
		t.Fatalf("unexpected comment links: %#v", comment.Content.Links)
	}
	if comment.CrawlKeyword != "bia tiger" {
		t.Fatalf("unexpected comment crawl_keyword: %s", comment.CrawlKeyword)
	}
}

func TestFlattenFacebookFullFlowVideoAttachment(t *testing.T) {
	raw := []byte(`{
		"posts": [
			{
				"post": {
					"post_id": "fb-2",
					"message": "Video post",
					"url": "https://www.facebook.com/posts/fb-2",
					"author": {
						"id": "author-2",
						"name": "Video Page",
						"url": "https://www.facebook.com/video-page",
						"avatar_url": "https://example.com/avatar2.jpg"
					},
					"created_time": 1711267201,
					"reaction_count": 20,
					"comment_count": 0,
					"share_count": 4,
					"attachments": [
						{
							"type": "Video",
							"url": "https://www.facebook.com/watch?v=fb-2",
							"media_url": null,
							"width": null,
							"height": null
						}
					]
				},
				"comments": {
					"post_id": "fb-2",
					"total_count": 0,
					"comments": []
				}
			}
		]
	}`)

	input := uap.ParseAndStoreRawBatchInput{
		ProjectID:      "project-1",
		TaskID:         "task-1",
		Platform:       "facebook",
		Action:         "full_flow",
		RequestPayload: json.RawMessage(`{"params":{"keyword":"bia saigon"}}`),
		CompletionTime: time.Date(2026, time.March, 24, 12, 0, 0, 0, time.UTC),
	}

	uc := &implUseCase{}
	records, err := uc.flattenFacebookFullFlow(raw, input, nil)
	if err != nil {
		t.Fatalf("flattenFacebookFullFlow() error = %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}

	post := records[0]
	if post.Content.Subtitle != "" {
		t.Fatalf("expected empty subtitle, got %q", post.Content.Subtitle)
	}
	if len(post.Media) != 1 {
		t.Fatalf("expected 1 media item, got %d", len(post.Media))
	}
	if post.Media[0].Type != "video" {
		t.Fatalf("unexpected media type: %s", post.Media[0].Type)
	}
	if post.Media[0].DownloadURL != "" {
		t.Fatalf("unexpected video download_url: %q", post.Media[0].DownloadURL)
	}
	if post.Media[0].Width != nil || post.Media[0].Height != nil {
		t.Fatalf("expected nil dimensions, got width=%#v height=%#v", post.Media[0].Width, post.Media[0].Height)
	}
	if post.CrawlKeyword != "bia saigon" {
		t.Fatalf("unexpected crawl_keyword on video post: %s", post.CrawlKeyword)
	}
}

func TestFlattenFacebookFullFlowDoesNotEmitReplyRecords(t *testing.T) {
	raw := []byte(`{
		"posts": [
			{
				"post": {
					"post_id": "fb-3",
					"message": "Threaded post",
					"url": "https://www.facebook.com/posts/fb-3",
					"author": {
						"id": "author-3",
						"name": "Author 3",
						"url": "https://www.facebook.com/author3",
						"avatar_url": "https://example.com/a3.jpg"
					},
					"created_time": 1711267202,
					"reaction_count": 1,
					"comment_count": 2,
					"share_count": 0,
					"attachments": null
				},
				"comments": {
					"comments": [
						{
							"id": "fc-2",
							"message": "Comment one",
							"author": {
								"id": "commenter-2",
								"name": "Commenter 2",
								"profile_url": "https://www.facebook.com/commenter2",
								"avatar_url": "https://example.com/c2.jpg"
							},
							"created_time": 1711267802,
							"reaction_count": 0,
							"reply_count": 5,
							"replies": null
						}
					]
				}
			}
		]
	}`)

	input := uap.ParseAndStoreRawBatchInput{
		ProjectID:      "project-1",
		TaskID:         "task-1",
		Platform:       "facebook",
		Action:         "full_flow",
		CompletionTime: time.Date(2026, time.March, 24, 12, 0, 0, 0, time.UTC),
	}

	uc := &implUseCase{}
	records, err := uc.flattenFacebookFullFlow(raw, input, nil)
	if err != nil {
		t.Fatalf("flattenFacebookFullFlow() error = %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(records))
	}
	for _, record := range records {
		if record.Identity.UAPType == uap.UAPTypeReply {
			t.Fatalf("unexpected REPLY record: %#v", record)
		}
	}
	comment := records[1]
	if comment.Engagement.ReplyCount == nil || *comment.Engagement.ReplyCount != 5 {
		t.Fatalf("unexpected reply_count: %#v", comment.Engagement.ReplyCount)
	}
	facebookMeta, ok := comment.PlatformMeta["facebook"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected comment platform_meta.facebook")
	}
	if facebookMeta["replies_present"] != false {
		t.Fatalf("unexpected replies_present: %#v", facebookMeta["replies_present"])
	}
}

func TestFlattenFacebookFullFlowToleratesMissingComments(t *testing.T) {
	raw := []byte(`{
		"result": {
			"posts": [
				{
					"post": {
						"post_id": "fb-4",
						"message": "No comments block",
						"url": "https://www.facebook.com/posts/fb-4",
						"author": {
							"id": "author-4",
							"name": "Author 4",
							"url": "https://www.facebook.com/author4",
							"avatar_url": "https://example.com/a4.jpg"
						},
						"created_time": 1711267203,
						"reaction_count": 11,
						"comment_count": 0,
						"share_count": 1,
						"attachments": null
					},
					"comments": {
						"error": "boom"
					}
				}
			]
		}
	}`)

	input := uap.ParseAndStoreRawBatchInput{
		ProjectID:      "project-1",
		TaskID:         "task-1",
		Platform:       "facebook",
		Action:         "full_flow",
		CompletionTime: time.Date(2026, time.March, 24, 12, 0, 0, 0, time.UTC),
	}

	uc := &implUseCase{}
	records, err := uc.flattenFacebookFullFlow(raw, input, nil)
	if err != nil {
		t.Fatalf("flattenFacebookFullFlow() error = %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	if records[0].Identity.UAPID != "fb_p_fb-4" {
		t.Fatalf("unexpected post uap_id: %s", records[0].Identity.UAPID)
	}
}
