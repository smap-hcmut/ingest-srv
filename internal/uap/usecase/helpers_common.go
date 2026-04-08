package usecase

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"ingest-srv/internal/uap"
	repo "ingest-srv/internal/uap/repository"
	"io"
	"net/http"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/smap-hcmut/shared-libs/go/minio"
)

const maxSubtitleBodyBytes = 1 << 20

var (
	webvttTimestampPattern      = regexp.MustCompile(`^\s*\d{2}:\d{2}:\d{2}\.\d{3}\s+-->\s+\d{2}:\d{2}:\d{2}\.\d{3}`)
	webvttTimestampShortPattern = regexp.MustCompile(`^\s*\d{2}:\d{2}\.\d{3}\s+-->\s+\d{2}:\d{2}\.\d{3}`)
	webvttCueIndexPattern       = regexp.MustCompile(`^\s*\d+\s*$`)
	linkPattern                 = regexp.MustCompile(`https?://[^\s<>"']+`)
	youtubeHandlePattern        = regexp.MustCompile(`youtube\.com/(@[A-Za-z0-9._-]+)`)
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
			"width":        item.Width,
			"height":       item.Height,
		})
	}

	platformMeta := record.PlatformMeta
	if platformMeta == nil {
		platformMeta = map[string]interface{}{}
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
			"text":     record.Content.Text,
			"title":    record.Content.Title,
			"subtitle": record.Content.Subtitle,
			"hashtags": record.Content.Hashtags,
			"keywords": record.Content.Keywords,
			"language": record.Content.Language,
			"links":    record.Content.Links,
		},
		"author": map[string]interface{}{
			"id":          record.Author.ID,
			"username":    record.Author.Username,
			"nickname":    record.Author.Nickname,
			"avatar":      record.Author.Avatar,
			"profile_url": record.Author.ProfileURL,
			"is_verified": record.Author.IsVerified,
		},
		"engagement": map[string]interface{}{
			"likes":          record.Engagement.Likes,
			"comments_count": record.Engagement.CommentsCount,
			"shares":         record.Engagement.Shares,
			"views":          record.Engagement.Views,
			"saves":          record.Engagement.Saves,
			"reply_count":    record.Engagement.ReplyCount,
		},
		"media": media,
		"temporal": map[string]interface{}{
			"posted_at":   record.Temporal.PostedAt,
			"updated_at":  record.Temporal.UpdatedAt,
			"ingested_at": record.Temporal.IngestedAt,
		},
		"domain_type_code": strings.TrimSpace(record.DomainTypeCode),
		"crawl_keyword":    strings.TrimSpace(record.CrawlKeyword),
		"platform_meta":    platformMeta,
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

func (uc *implUseCase) extractCrawlKeyword(requestPayload json.RawMessage) string {
	if len(requestPayload) == 0 {
		return ""
	}

	var root map[string]interface{}
	if err := json.Unmarshal(requestPayload, &root); err != nil {
		return ""
	}

	params := uc.toMap(root["params"])
	return strings.TrimSpace(uc.stringAt(params, "keyword"))
}

func (uc *implUseCase) resolveSubtitleText(urls ...string) string {
	seen := make(map[string]struct{}, len(urls))
	for _, rawURL := range urls {
		url := strings.TrimSpace(rawURL)
		if url == "" {
			continue
		}
		if _, ok := seen[url]; ok {
			continue
		}
		seen[url] = struct{}{}

		text, err := uc.downloadSubtitleText(url)
		if err != nil {
			if uc.l != nil {
				uc.l.Warnf(context.Background(), "uap.usecase.resolveSubtitleText.download_failed: url=%s err=%v", url, err)
			}
			continue
		}
		if strings.TrimSpace(text) != "" {
			return text
		}
	}

	return ""
}

func (uc *implUseCase) downloadSubtitleText(url string) (string, error) {
	url = strings.TrimSpace(url)
	if url == "" {
		return "", errors.New("empty subtitle url")
	}

	client := uc.subtitleHTTPClient
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", errors.New(resp.Status)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxSubtitleBodyBytes+1))
	if err != nil {
		return "", err
	}
	if len(body) > maxSubtitleBodyBytes {
		return "", errors.New("subtitle body exceeds max size")
	}

	if strings.Contains(strings.ToUpper(string(body)), "WEBVTT") || strings.Contains(string(body), "-->") {
		return uc.normalizeWEBVTT(body), nil
	}

	return uc.normalizeTranscriptText(string(body)), nil
}

func (uc *implUseCase) normalizeWEBVTT(body []byte) string {
	text := strings.ReplaceAll(string(body), "\r\n", "\n")
	text = strings.TrimPrefix(text, "\uFEFF")

	lines := strings.Split(text, "\n")
	parts := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.EqualFold(trimmed, "WEBVTT") {
			continue
		}
		if webvttCueIndexPattern.MatchString(trimmed) {
			continue
		}
		if webvttTimestampPattern.MatchString(trimmed) || webvttTimestampShortPattern.MatchString(trimmed) {
			continue
		}
		parts = append(parts, trimmed)
	}

	return uc.normalizeTranscriptText(strings.Join(parts, " "))
}

func (uc *implUseCase) normalizeTranscriptText(s string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(s)), " ")
}

func (uc *implUseCase) extractLinks(texts ...string) []string {
	seen := make(map[string]struct{})
	links := make([]string, 0)
	for _, text := range texts {
		for _, match := range linkPattern.FindAllString(text, -1) {
			link := strings.TrimSpace(strings.TrimRight(match, ".,);!?:]}'\""))
			if link == "" {
				continue
			}
			if _, ok := seen[link]; ok {
				continue
			}
			seen[link] = struct{}{}
			links = append(links, link)
		}
	}
	if len(links) == 0 {
		return nil
	}
	return links
}

func (uc *implUseCase) extractYouTubeHandle(rawURL string) string {
	matches := youtubeHandlePattern.FindStringSubmatch(strings.TrimSpace(rawURL))
	if len(matches) < 2 {
		return ""
	}
	return strings.TrimSpace(matches[1])
}

func (uc *implUseCase) buildYouTubeChannelURL(channelID string) string {
	channelID = strings.TrimSpace(channelID)
	if channelID == "" {
		return ""
	}
	return "https://www.youtube.com/channel/" + channelID
}

func (uc *implUseCase) joinTranscriptSegments(segments []uap.YouTubeTranscriptSegmentInput) string {
	if len(segments) == 0 {
		return ""
	}
	parts := make([]string, 0, len(segments))
	for _, segment := range segments {
		text := strings.TrimSpace(segment.Text)
		if text == "" {
			continue
		}
		parts = append(parts, text)
	}
	if len(parts) == 0 {
		return ""
	}
	return uc.normalizeTranscriptText(strings.Join(parts, " "))
}

func (uc *implUseCase) parseDurationText(raw string) *int {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}

	parts := strings.Split(raw, ":")
	if len(parts) != 2 && len(parts) != 3 {
		return nil
	}

	total := 0
	multiplier := 1
	for i := len(parts) - 1; i >= 0; i-- {
		value, err := strconv.Atoi(strings.TrimSpace(parts[i]))
		if err != nil || value < 0 {
			return nil
		}
		total += value * multiplier
		multiplier *= 60
	}

	return uc.intPtr(total)
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

func (uc *implUseCase) intPtrIfPositive(v int) *int {
	if v <= 0 {
		return nil
	}
	return &v
}

func (uc *implUseCase) float64Ptr(v float64) *float64 {
	return &v
}

func (uc *implUseCase) readAllAndClose(reader io.ReadCloser) ([]byte, error) {
	defer reader.Close()
	return io.ReadAll(reader)
}
