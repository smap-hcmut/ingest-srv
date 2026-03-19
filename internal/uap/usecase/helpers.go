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
	"time"

	"github.com/smap-hcmut/shared-libs/go/minio"
)

func (uc *implUseCase) validateParseAndStoreRawBatchInput(input uap.ParseAndStoreRawBatchInput) error {
	if strings.TrimSpace(input.RawBatchID) == "" ||
		strings.TrimSpace(input.ProjectID) == "" ||
		strings.TrimSpace(input.SourceID) == "" ||
		strings.TrimSpace(input.TaskID) == "" ||
		strings.TrimSpace(input.StorageBucket) == "" ||
		strings.TrimSpace(input.StoragePath) == "" ||
		strings.TrimSpace(input.BatchID) == "" {
		return uap.ErrInvalidRawBatchInput
	}

	if !strings.EqualFold(strings.TrimSpace(input.Platform), uap.PlatformTikTok) ||
		!strings.EqualFold(strings.TrimSpace(input.Action), uap.TaskTypeFullFlow) {
		return uap.ErrInvalidRawBatchInput
	}

	return nil
}

func (uc *implUseCase) flattenTikTokFullFlow(rawBytes []byte, input uap.ParseAndStoreRawBatchInput, onRecord func(uap.UAPRecord)) ([]uap.UAPRecord, error) {
	flattenInput, err := uc.parseTikTokFullFlowInput(rawBytes)
	if err != nil {
		return nil, err
	}

	records := make([]uap.UAPRecord, 0)
	for _, bundle := range flattenInput.Posts {
		postRecord, rootID := uc.mapTikTokPost(bundle, input)
		if rootID == "" {
			continue
		}
		records = append(records, postRecord)
		if onRecord != nil {
			onRecord(postRecord)
		}

		for _, comment := range bundle.Comments {
			commentRecord, commentID := uc.mapTikTokComment(comment, input, rootID)
			if commentID == "" {
				continue
			}
			records = append(records, commentRecord)
			if onRecord != nil {
				onRecord(commentRecord)
			}

			for _, reply := range comment.ReplyComments {
				replyRecord := uc.mapTikTokReply(reply, input, rootID, commentID)
				if replyRecord.Identity.OriginID == "" {
					continue
				}
				records = append(records, replyRecord)
				if onRecord != nil {
					onRecord(replyRecord)
				}
			}
		}
	}

	return records, nil
}

func (uc *implUseCase) parseTikTokFullFlowInput(rawBytes []byte) (uap.TikTokFullFlowInput, error) {
	var root map[string]interface{}
	if err := json.Unmarshal(rawBytes, &root); err != nil {
		return uap.TikTokFullFlowInput{}, uap.ErrParseRawPayload
	}

	result := uc.toMap(root["result"])
	postItems := uc.toSlice(result["posts"])
	posts := make([]uap.TikTokPostBundleInput, 0, len(postItems))
	for _, postItem := range postItems {
		bundleMap := uc.toMap(postItem)
		postMap := uc.toMap(bundleMap["post"])
		detailMap := uc.toMap(bundleMap["detail"])
		commentsRoot := uc.toMap(bundleMap["comments"])
		commentItems := uc.toSlice(commentsRoot["comments"])

		comments := make([]uap.TikTokCommentInput, 0, len(commentItems))
		for _, commentItem := range commentItems {
			commentMap := uc.toMap(commentItem)
			replyItems := uc.toSlice(commentMap["reply_comments"])
			replies := make([]uap.TikTokReplyInput, 0, len(replyItems))
			for _, replyItem := range replyItems {
				replyMap := uc.toMap(replyItem)
				replyAuthor := uc.toMap(replyMap["author"])
				replies = append(replies, uap.TikTokReplyInput{
					ReplyID:    uc.stringAt(replyMap, "reply_id"),
					Content:    uc.stringAt(replyMap, "content"),
					LikesCount: uc.intAt(replyMap, "likes_count"),
					RepliedAt:  uc.stringAt(replyMap, "replied_at"),
					Author: uap.TikTokAuthorInput{
						UID:      uc.stringAt(replyAuthor, "uid"),
						Username: uc.stringAt(replyAuthor, "username"),
						Nickname: uc.stringAt(replyAuthor, "nickname"),
						Avatar:   uc.stringAt(replyAuthor, "avatar"),
					},
				})
			}

			commentAuthor := uc.toMap(commentMap["author"])
			scoreMap := uc.toMap(commentMap["sort_extra_score"])
			comments = append(comments, uap.TikTokCommentInput{
				CommentID:   uc.stringAt(commentMap, "comment_id"),
				Content:     uc.stringAt(commentMap, "content"),
				LikesCount:  uc.intAt(commentMap, "likes_count"),
				ReplyCount:  uc.intAt(commentMap, "reply_count"),
				CommentedAt: uc.stringAt(commentMap, "commented_at"),
				SortExtraScore: uap.TikTokCommentScoreInput{
					ReplyScore:    uc.floatAt(scoreMap, "reply_score"),
					ShowMoreScore: uc.floatAt(scoreMap, "show_more_score"),
				},
				ReplyComments: replies,
				Author: uap.TikTokAuthorInput{
					UID:      uc.stringAt(commentAuthor, "uid"),
					Username: uc.stringAt(commentAuthor, "username"),
					Nickname: uc.stringAt(commentAuthor, "nickname"),
					Avatar:   uc.stringAt(commentAuthor, "avatar"),
				},
			})
		}

		postAuthor := uc.toMap(postMap["author"])
		detailAuthor := uc.toMap(detailMap["author"])
		detailSummary := uc.toMap(detailMap["summary"])
		detailDownloads := uc.toMap(detailMap["downloads"])

		posts = append(posts, uap.TikTokPostBundleInput{
			Post: uap.TikTokPostInput{
				VideoID:       uc.stringAt(postMap, "video_id"),
				URL:           uc.stringAt(postMap, "url"),
				Description:   uc.stringAt(postMap, "description"),
				LikesCount:    uc.intAt(postMap, "likes_count"),
				CommentsCount: uc.intAt(postMap, "comments_count"),
				SharesCount:   uc.intAt(postMap, "shares_count"),
				ViewsCount:    uc.intAt(postMap, "views_count"),
				Hashtags:      uc.stringSliceAt(postMap, "hashtags"),
				PostedAt:      uc.stringAt(postMap, "posted_at"),
				IsShopVideo:   uc.boolAt(postMap, "is_shop_video"),
				Author: uap.TikTokAuthorInput{
					UID:      uc.stringAt(postAuthor, "uid"),
					Username: uc.stringAt(postAuthor, "username"),
					Nickname: uc.stringAt(postAuthor, "nickname"),
					Avatar:   uc.stringAt(postAuthor, "avatar"),
				},
			},
			Detail: uap.TikTokDetailInput{
				VideoID:        uc.stringAt(detailMap, "video_id"),
				URL:            uc.stringAt(detailMap, "url"),
				Description:    uc.stringAt(detailMap, "description"),
				LikesCount:     uc.intAt(detailMap, "likes_count"),
				CommentsCount:  uc.intAt(detailMap, "comments_count"),
				SharesCount:    uc.intAt(detailMap, "shares_count"),
				ViewsCount:     uc.intAt(detailMap, "views_count"),
				BookmarksCount: uc.intAt(detailMap, "bookmarks_count"),
				Hashtags:       uc.stringSliceAt(detailMap, "hashtags"),
				MusicTitle:     uc.stringAt(detailMap, "music_title"),
				MusicURL:       uc.stringAt(detailMap, "music_url"),
				Duration:       uc.intAt(detailMap, "duration"),
				PostedAt:       uc.stringAt(detailMap, "posted_at"),
				IsShopVideo:    uc.boolAt(detailMap, "is_shop_video"),
				PlayURL:        uc.stringAt(detailMap, "play_url"),
				DownloadURL:    uc.stringAt(detailMap, "download_url"),
				CoverURL:       uc.stringAt(detailMap, "cover_url"),
				OriginCoverURL: uc.stringAt(detailMap, "origin_cover_url"),
				SubtitleURL:    uc.stringAt(detailMap, "subtitle_url"),
				Downloads: uap.TikTokDetailAssetsInput{
					Music:    uc.stringAt(detailDownloads, "music"),
					Cover:    uc.stringAt(detailDownloads, "cover"),
					Subtitle: uc.stringAt(detailDownloads, "subtitle"),
					Video:    uc.stringAt(detailDownloads, "video"),
				},
				Summary: uap.TikTokSummaryInput{
					Title:    uc.stringAt(detailSummary, "title"),
					Keywords: uc.stringSliceAt(detailSummary, "keywords"),
					Language: uc.stringAt(detailSummary, "language"),
				},
				Author: uap.TikTokAuthorInput{
					UID:      uc.stringAt(detailAuthor, "uid"),
					Username: uc.stringAt(detailAuthor, "username"),
					Nickname: uc.stringAt(detailAuthor, "nickname"),
					Avatar:   uc.stringAt(detailAuthor, "avatar"),
				},
			},
			Comments: comments,
		})
	}

	return uap.TikTokFullFlowInput{Posts: posts}, nil
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

func (uc *implUseCase) mapTikTokPost(bundle uap.TikTokPostBundleInput, input uap.ParseAndStoreRawBatchInput) (uap.UAPRecord, string) {
	videoID := strings.TrimSpace(uc.firstNonEmpty(bundle.Detail.VideoID, bundle.Post.VideoID))
	if videoID == "" {
		return uap.UAPRecord{}, ""
	}

	rootID := uc.buildUAPID("tt_p_", videoID)
	isVerified := false
	isShopVideo := bundle.Detail.IsShopVideo || bundle.Post.IsShopVideo
	postedAt := strings.TrimSpace(uc.firstNonEmpty(bundle.Detail.PostedAt, bundle.Post.PostedAt))
	ingestedAt := input.CompletionTime.UTC().Format(time.RFC3339)

	return uap.UAPRecord{
		Identity: uap.UAPIdentity{
			UAPID:     rootID,
			OriginID:  videoID,
			UAPType:   uap.UAPTypePost,
			Platform:  uap.PlatformTikTok,
			URL:       uc.firstNonEmpty(bundle.Detail.URL, bundle.Post.URL),
			TaskID:    input.TaskID,
			ProjectID: input.ProjectID,
		},
		Hierarchy: uap.UAPHierarchy{
			ParentID: nil,
			RootID:   rootID,
			Depth:    0,
		},
		Content: uap.UAPContent{
			Text:           uc.firstNonEmpty(bundle.Detail.Description, bundle.Post.Description),
			Hashtags:       uc.normalizeStringSlice(uc.firstNonEmptySlice(bundle.Detail.Hashtags, bundle.Post.Hashtags)),
			TikTokKeywords: uc.normalizeStringSlice(bundle.Detail.Summary.Keywords),
			IsShopVideo:    &isShopVideo,
			MusicTitle:     bundle.Detail.MusicTitle,
			MusicURL:       uc.firstNonEmpty(bundle.Detail.Downloads.Music, bundle.Detail.MusicURL),
			SummaryTitle:   bundle.Detail.Summary.Title,
			SubtitleURL:    uc.firstNonEmpty(bundle.Detail.Downloads.Subtitle, bundle.Detail.SubtitleURL),
			Language:       strings.TrimSpace(bundle.Detail.Summary.Language),
		},
		Author: uap.UAPAuthor{
			ID:         uc.firstNonEmpty(bundle.Detail.Author.UID, bundle.Post.Author.UID),
			Username:   uc.firstNonEmpty(bundle.Detail.Author.Username, bundle.Post.Author.Username),
			Nickname:   uc.firstNonEmpty(bundle.Detail.Author.Nickname, bundle.Post.Author.Nickname),
			Avatar:     uc.firstNonEmpty(bundle.Detail.Author.Avatar, bundle.Post.Author.Avatar),
			IsVerified: &isVerified,
		},
		Engagement: uap.UAPEngagement{
			Likes:         uc.intPtr(uc.firstNonZero(bundle.Detail.LikesCount, bundle.Post.LikesCount)),
			CommentsCount: uc.intPtr(uc.firstNonZero(bundle.Detail.CommentsCount, bundle.Post.CommentsCount)),
			Shares:        uc.intPtr(uc.firstNonZero(bundle.Detail.SharesCount, bundle.Post.SharesCount)),
			Views:         uc.intPtr(uc.firstNonZero(bundle.Detail.ViewsCount, bundle.Post.ViewsCount)),
			Bookmarks:     uc.intPtr(bundle.Detail.BookmarksCount),
		},
		Media: []uap.UAPMedia{
			{
				Type:        "video",
				URL:         uc.firstNonEmpty(bundle.Detail.PlayURL, bundle.Post.URL),
				DownloadURL: uc.firstNonEmpty(bundle.Detail.Downloads.Video, bundle.Detail.DownloadURL),
				Duration:    uc.intPtr(bundle.Detail.Duration),
				Thumbnail:   uc.firstNonEmpty(bundle.Detail.Downloads.Cover, bundle.Detail.CoverURL, bundle.Detail.OriginCoverURL),
			},
		},
		Temporal: uap.UAPTemporal{
			PostedAt:   postedAt,
			IngestedAt: ingestedAt,
		},
	}, rootID
}

func (uc *implUseCase) mapTikTokComment(comment uap.TikTokCommentInput, input uap.ParseAndStoreRawBatchInput, rootID string) (uap.UAPRecord, string) {
	commentID := strings.TrimSpace(comment.CommentID)
	if commentID == "" {
		return uap.UAPRecord{}, ""
	}

	parentID := rootID
	sortScore := comment.SortExtraScore.ShowMoreScore
	if sortScore == 0 {
		sortScore = comment.SortExtraScore.ReplyScore
	}

	return uap.UAPRecord{
		Identity: uap.UAPIdentity{
			UAPID:     uc.buildUAPID("tt_c_", commentID),
			OriginID:  commentID,
			UAPType:   uap.UAPTypeComment,
			Platform:  uap.PlatformTikTok,
			TaskID:    input.TaskID,
			ProjectID: input.ProjectID,
		},
		Hierarchy: uap.UAPHierarchy{
			ParentID: &parentID,
			RootID:   rootID,
			Depth:    1,
		},
		Content: uap.UAPContent{
			Text: strings.TrimSpace(comment.Content),
		},
		Author: uap.UAPAuthor{
			ID:       comment.Author.UID,
			Username: comment.Author.Username,
			Nickname: comment.Author.Nickname,
			Avatar:   comment.Author.Avatar,
		},
		Engagement: uap.UAPEngagement{
			Likes:      uc.intPtr(comment.LikesCount),
			ReplyCount: uc.intPtr(comment.ReplyCount),
			SortScore:  uc.float64Ptr(sortScore),
		},
		Temporal: uap.UAPTemporal{
			PostedAt: strings.TrimSpace(comment.CommentedAt),
		},
	}, commentID
}

func (uc *implUseCase) mapTikTokReply(reply uap.TikTokReplyInput, input uap.ParseAndStoreRawBatchInput, rootID, commentID string) uap.UAPRecord {
	replyID := strings.TrimSpace(reply.ReplyID)
	if replyID == "" {
		return uap.UAPRecord{}
	}

	parentID := uc.buildUAPID("tt_c_", commentID)

	return uap.UAPRecord{
		Identity: uap.UAPIdentity{
			UAPID:     uc.buildUAPID("tt_r_", replyID),
			OriginID:  replyID,
			UAPType:   uap.UAPTypeReply,
			Platform:  uap.PlatformTikTok,
			TaskID:    input.TaskID,
			ProjectID: input.ProjectID,
		},
		Hierarchy: uap.UAPHierarchy{
			ParentID: &parentID,
			RootID:   rootID,
			Depth:    2,
		},
		Content: uap.UAPContent{
			Text: strings.TrimSpace(reply.Content),
		},
		Author: uap.UAPAuthor{
			ID:       reply.Author.UID,
			Username: reply.Author.Username,
			Nickname: reply.Author.Nickname,
			Avatar:   reply.Author.Avatar,
		},
		Engagement: uap.UAPEngagement{
			Likes: uc.intPtr(reply.LikesCount),
		},
		Temporal: uap.UAPTemporal{
			PostedAt: strings.TrimSpace(reply.RepliedAt),
		},
	}
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
