package usecase

import (
	"context"
	"encoding/json"
	"testing"

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
		Content: uap.UAPContent{Text: "video text", Language: "vi", TikTokKeywords: []string{"vinfast"}},
		Author: uap.UAPAuthor{
			ID:       "author-1",
			Username: "demo_author",
			Nickname: "Demo Author",
		},
		Engagement: uap.UAPEngagement{
			Likes:         ptrInt(10),
			CommentsCount: ptrInt(5),
			Shares:        ptrInt(2),
			Views:         ptrInt(100),
			Bookmarks:     ptrInt(3),
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
	}

	input := uap.ParseAndStoreRawBatchInput{
		RawBatchID: "raw-batch-1",
	}

	spy := &spyPublisher{}
	stats := &uap.KafkaPublishStats{Topic: "smap.collector.output"}
	uc := &implUseCase{publisher: spy, publishTopic: "smap.collector.output"}

	uc.publishRecord(context.Background(), record, input, stats)

	if stats.AttemptedCount != 1 || stats.SuccessCount != 1 || stats.FailedCount != 0 {
		t.Fatalf("unexpected publish stats: %#v", stats)
	}
	if spy.lastInput.Record.Identity.UAPID != record.Identity.UAPID {
		t.Fatalf("unexpected published uap_id: %s", spy.lastInput.Record.Identity.UAPID)
	}
	if spy.lastInput.Record.Content.Text != record.Content.Text {
		t.Fatalf("unexpected published payload text: %s", spy.lastInput.Record.Content.Text)
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
