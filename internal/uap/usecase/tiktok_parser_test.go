package usecase

import (
	"testing"
	"time"

	"ingest-srv/internal/uap"
)

func TestFlattenTikTokFullFlow(t *testing.T) {
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
						"subtitle_url": "https://example.com/subtitle.vtt",
						"downloads": {
							"music": "https://example.com/music-download.mp3",
							"cover": "https://example.com/cover-download.jpg",
							"subtitle": "https://example.com/subtitle-download.vtt",
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
		CompletionTime: time.Date(2026, time.March, 8, 4, 0, 0, 0, time.UTC),
	}

	uc := &implUseCase{}
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
}
