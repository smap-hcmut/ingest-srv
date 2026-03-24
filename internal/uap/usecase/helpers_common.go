package usecase

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"ingest-srv/internal/uap"
	repo "ingest-srv/internal/uap/repository"
	"io"
	"path"
	"strconv"
	"strings"

	"github.com/smap-hcmut/shared-libs/go/minio"
)

func (uc *implUseCase) validateParseAndStoreRawBatchInputCommon(input uap.ParseAndStoreRawBatchInput) error {
	if strings.TrimSpace(input.RawBatchID) == "" ||
		strings.TrimSpace(input.ProjectID) == "" ||
		strings.TrimSpace(input.SourceID) == "" ||
		strings.TrimSpace(input.TaskID) == "" ||
		strings.TrimSpace(input.StorageBucket) == "" ||
		strings.TrimSpace(input.StoragePath) == "" ||
		strings.TrimSpace(input.BatchID) == "" {
		return uap.ErrInvalidRawBatchInput
	}

	return nil
}

func (uc *implUseCase) toMap(v interface{}) map[string]interface{} {
	if m, ok := v.(map[string]interface{}); ok {
		return m
	}
	return map[string]interface{}{}
}

func (uc *implUseCase) toSlice(v interface{}) []interface{} {
	if s, ok := v.([]interface{}); ok {
		return s
	}
	return nil
}

func (uc *implUseCase) stringAt(m map[string]interface{}, key string) string {
	if raw, ok := m[key]; ok {
		switch v := raw.(type) {
		case string:
			return v
		}
	}
	return ""
}

func (uc *implUseCase) boolAt(m map[string]interface{}, key string) bool {
	if raw, ok := m[key]; ok {
		switch v := raw.(type) {
		case bool:
			return v
		case float64:
			return v != 0
		}
	}
	return false
}

func (uc *implUseCase) intAt(m map[string]interface{}, key string) int {
	if raw, ok := m[key]; ok {
		switch v := raw.(type) {
		case int:
			return v
		case int32:
			return int(v)
		case int64:
			return int(v)
		case float64:
			return int(v)
		case float32:
			return int(v)
		}
	}
	return 0
}

func (uc *implUseCase) floatAt(m map[string]interface{}, key string) float64 {
	if raw, ok := m[key]; ok {
		switch v := raw.(type) {
		case float64:
			return v
		case float32:
			return float64(v)
		case int:
			return float64(v)
		}
	}
	return 0
}

func (uc *implUseCase) stringSliceAt(m map[string]interface{}, key string) []string {
	raw, ok := m[key]
	if !ok {
		return nil
	}
	items, ok := raw.([]interface{})
	if !ok {
		return nil
	}
	values := make([]string, 0, len(items))
	for _, item := range items {
		if value, ok := item.(string); ok {
			values = append(values, value)
		}
	}
	return values
}

func (uc *implUseCase) chunkRecords(records []uap.UAPRecord) [][]uap.UAPRecord {
	if len(records) == 0 {
		return nil
	}

	chunks := make([][]uap.UAPRecord, 0, (len(records)+uap.ChunkSize-1)/uap.ChunkSize)
	for start := 0; start < len(records); start += uap.ChunkSize {
		end := start + uap.ChunkSize
		if end > len(records) {
			end = len(records)
		}
		chunks = append(chunks, records[start:end])
	}

	return chunks
}

func (uc *implUseCase) marshalChunkJSONL(records []uap.UAPRecord) ([]byte, error) {
	var buf bytes.Buffer
	for _, record := range records {
		body, err := uc.marshalUAPRecord(record)
		if err != nil {
			return nil, err
		}
		buf.Write(body)
		buf.WriteByte('\n')
	}
	return buf.Bytes(), nil
}

func (uc *implUseCase) buildPartPath(projectID, sourceID, batchID string, partNo int) string {
	return path.Join(
		"uap-batches",
		strings.TrimSpace(projectID),
		strings.TrimSpace(sourceID),
		strings.TrimSpace(batchID),
		fmt.Sprintf("part-%05d.jsonl", partNo),
	)
}

func (uc *implUseCase) uploadChunk(ctx context.Context, client minio.MinIO, bucket, projectID, sourceID, batchID string, partNo int, records []uap.UAPRecord) (uap.ArtifactPart, error) {
	payload, err := uc.marshalChunkJSONL(records)
	if err != nil {
		return uap.ArtifactPart{}, err
	}

	objectPath := uc.buildPartPath(projectID, sourceID, batchID, partNo)
	reader := bytes.NewReader(payload)
	if _, err := client.UploadFile(ctx, &minio.UploadRequest{
		BucketName:   bucket,
		ObjectName:   objectPath,
		OriginalName: path.Base(objectPath),
		Reader:       reader,
		Size:         int64(len(payload)),
		ContentType:  uap.ContentTypeNDJSON,
		Metadata: map[string]string{
			"project-id": projectID,
			"source-id":  sourceID,
			"batch-id":   batchID,
			"part-no":    strconv.Itoa(partNo),
		},
	}); err != nil {
		return uap.ArtifactPart{}, err
	}

	return uap.ArtifactPart{
		PartNo:        partNo,
		StorageBucket: bucket,
		StoragePath:   objectPath,
		RecordCount:   len(records),
	}, nil
}

func (uc *implUseCase) marshalUAPRecord(record uap.UAPRecord) ([]byte, error) {
	media := make([]map[string]interface{}, 0, len(record.Media))
	for _, item := range record.Media {
		media = append(media, map[string]interface{}{
			"type":         item.Type,
			"url":          item.URL,
			"download_url": item.DownloadURL,
			"duration":     item.Duration,
			"thumbnail":    item.Thumbnail,
		})
	}

	payload := map[string]interface{}{
		"identity": map[string]interface{}{
			"uap_id":     record.Identity.UAPID,
			"origin_id":  record.Identity.OriginID,
			"uap_type":   string(record.Identity.UAPType),
			"platform":   record.Identity.Platform,
			"url":        record.Identity.URL,
			"task_id":    record.Identity.TaskID,
			"project_id": record.Identity.ProjectID,
		},
		"hierarchy": map[string]interface{}{
			"parent_id": record.Hierarchy.ParentID,
			"root_id":   record.Hierarchy.RootID,
			"depth":     record.Hierarchy.Depth,
		},
		"content": map[string]interface{}{
			"text":            record.Content.Text,
			"hashtags":        record.Content.Hashtags,
			"tiktok_keywords": record.Content.TikTokKeywords,
			"is_shop_video":   record.Content.IsShopVideo,
			"music_title":     record.Content.MusicTitle,
			"music_url":       record.Content.MusicURL,
			"summary_title":   record.Content.SummaryTitle,
			"subtitle_url":    record.Content.SubtitleURL,
			"language":        record.Content.Language,
			"external_links":  record.Content.ExternalLinks,
		},
		"author": map[string]interface{}{
			"id":          record.Author.ID,
			"username":    record.Author.Username,
			"nickname":    record.Author.Nickname,
			"avatar":      record.Author.Avatar,
			"is_verified": record.Author.IsVerified,
		},
		"engagement": map[string]interface{}{
			"likes":          record.Engagement.Likes,
			"comments_count": record.Engagement.CommentsCount,
			"shares":         record.Engagement.Shares,
			"views":          record.Engagement.Views,
			"bookmarks":      record.Engagement.Bookmarks,
			"reply_count":    record.Engagement.ReplyCount,
			"sort_score":     record.Engagement.SortScore,
		},
		"media": media,
		"temporal": map[string]interface{}{
			"posted_at":   record.Temporal.PostedAt,
			"ingested_at": record.Temporal.IngestedAt,
		},
	}

	return json.Marshal(payload)
}

func (uc *implUseCase) normalizeStringSlice(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	result := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		result = append(result, trimmed)
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func (uc *implUseCase) firstNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func (uc *implUseCase) firstNonEmptySlice(values ...[]string) []string {
	for _, value := range values {
		if normalized := uc.normalizeStringSlice(value); len(normalized) > 0 {
			return normalized
		}
	}
	return nil
}

func (uc *implUseCase) firstNonZero(values ...int) int {
	for _, value := range values {
		if value != 0 {
			return value
		}
	}
	return 0
}

func (uc *implUseCase) buildUAPID(prefix, originID string) string {
	return prefix + strings.TrimSpace(originID)
}

func (uc *implUseCase) mergeRawMetadata(existing json.RawMessage, parts []uap.ArtifactPart, totalRecords int, publishStats *uap.KafkaPublishStats) (json.RawMessage, error) {
	root := make(map[string]interface{})
	if len(existing) > 0 {
		if err := json.Unmarshal(existing, &root); err != nil {
			root = make(map[string]interface{})
		}
	}

	partPayload := make([]map[string]interface{}, 0, len(parts))
	for _, part := range parts {
		partPayload = append(partPayload, map[string]interface{}{
			"part_no":        part.PartNo,
			"storage_bucket": part.StorageBucket,
			"storage_path":   part.StoragePath,
			"record_count":   part.RecordCount,
		})
	}

	root[uap.ArtifactsMetadataKey] = map[string]interface{}{
		"version":       uap.ArtifactsVersionV1,
		"chunk_size":    uap.ChunkSize,
		"total_records": totalRecords,
		"total_parts":   len(parts),
		"content_type":  uap.ContentTypeNDJSON,
		"parts":         partPayload,
	}

	if publishStats != nil {
		root[uap.KafkaPublishKey] = map[string]interface{}{
			"topic":           strings.TrimSpace(publishStats.Topic),
			"attempted_count": publishStats.AttemptedCount,
			"success_count":   publishStats.SuccessCount,
			"failed_count":    publishStats.FailedCount,
			"last_error":      strings.TrimSpace(publishStats.LastError),
		}
	}

	return json.Marshal(root)
}

func (uc *implUseCase) failRawBatch(
	ctx context.Context,
	input uap.ParseAndStoreRawBatchInput,
	errMessage, publishErr string,
	parts []uap.ArtifactPart,
	totalRecords int,
	publishStats *uap.KafkaPublishStats,
) error {
	metadata, _ := uc.mergeRawMetadata(input.RawMetadata, parts, totalRecords, publishStats)
	if err := uc.repo.MarkRawBatchFailed(ctx, repo.MarkRawBatchFailedOptions{
		RawBatchID:   input.RawBatchID,
		ErrorMessage: errMessage,
		PublishError: publishErr,
		RawMetadata:  metadata,
	}); err != nil {
		uc.l.Errorf(ctx, "uap.usecase.failRawBatch.MarkRawBatchFailed: %v", err)
		return err
	}

	return nil
}

func (uc *implUseCase) intPtr(v int) *int {
	return &v
}

func (uc *implUseCase) float64Ptr(v float64) *float64 {
	return &v
}

func (uc *implUseCase) readAllAndClose(reader io.ReadCloser) ([]byte, error) {
	defer reader.Close()
	return io.ReadAll(reader)
}
