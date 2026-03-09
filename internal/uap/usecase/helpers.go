package usecase

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"path"
	"strconv"
	"strings"

	"ingest-srv/internal/uap"
	repo "ingest-srv/internal/uap/repository"
	minioPkg "ingest-srv/pkg/minio"
)

type artifactPart struct {
	PartNo      int    `json:"part_no"`
	StorageBucket string `json:"storage_bucket"`
	StoragePath string `json:"storage_path"`
	RecordCount int    `json:"record_count"`
}

type rawTikTokFullFlowEnvelope struct {
	Result rawTikTokFullFlowResult `json:"result"`
}

type rawTikTokFullFlowResult struct {
	Posts []rawTikTokFullFlowPostBundle `json:"posts"`
}

type rawTikTokFullFlowPostBundle struct {
	Post     rawTikTokPost     `json:"post"`
	Detail   rawTikTokDetail   `json:"detail"`
	Comments rawTikTokComments `json:"comments"`
}

type rawTikTokPost struct {
	VideoID       string           `json:"video_id"`
	URL           string           `json:"url"`
	Description   string           `json:"description"`
	Author        rawTikTokAuthor  `json:"author"`
	LikesCount    int              `json:"likes_count"`
	CommentsCount int              `json:"comments_count"`
	SharesCount   int              `json:"shares_count"`
	ViewsCount    int              `json:"views_count"`
	Hashtags      []string         `json:"hashtags"`
	PostedAt      string           `json:"posted_at"`
	IsShopVideo   bool             `json:"is_shop_video"`
}

type rawTikTokDetail struct {
	VideoID        string                `json:"video_id"`
	URL            string                `json:"url"`
	Description    string                `json:"description"`
	Author         rawTikTokAuthor       `json:"author"`
	LikesCount     int                   `json:"likes_count"`
	CommentsCount  int                   `json:"comments_count"`
	SharesCount    int                   `json:"shares_count"`
	ViewsCount     int                   `json:"views_count"`
	BookmarksCount int                   `json:"bookmarks_count"`
	Hashtags       []string              `json:"hashtags"`
	MusicTitle     string                `json:"music_title"`
	MusicURL       string                `json:"music_url"`
	Duration       int                   `json:"duration"`
	PostedAt       string                `json:"posted_at"`
	IsShopVideo    bool                  `json:"is_shop_video"`
	Summary        rawTikTokSummary      `json:"summary"`
	PlayURL        string                `json:"play_url"`
	DownloadURL    string                `json:"download_url"`
	CoverURL       string                `json:"cover_url"`
	OriginCoverURL string                `json:"origin_cover_url"`
	SubtitleURL    string                `json:"subtitle_url"`
	Downloads      rawTikTokDetailAssets `json:"downloads"`
}

type rawTikTokSummary struct {
	Title    string   `json:"title"`
	Keywords []string `json:"keywords"`
	Language string   `json:"language"`
}

type rawTikTokDetailAssets struct {
	Music    string `json:"music"`
	Cover    string `json:"cover"`
	Subtitle string `json:"subtitle"`
	Video    string `json:"video"`
}

type rawTikTokComments struct {
	Comments []rawTikTokComment `json:"comments"`
}

type rawTikTokComment struct {
	CommentID      string                  `json:"comment_id"`
	Content        string                  `json:"content"`
	Author         rawTikTokAuthor         `json:"author"`
	LikesCount     int                     `json:"likes_count"`
	ReplyCount     int                     `json:"reply_count"`
	SortExtraScore rawTikTokCommentScore   `json:"sort_extra_score"`
	ReplyComments  []rawTikTokReplyComment `json:"reply_comments"`
	CommentedAt    string                  `json:"commented_at"`
}

type rawTikTokCommentScore struct {
	ReplyScore    float64 `json:"reply_score"`
	ShowMoreScore float64 `json:"show_more_score"`
}

type rawTikTokReplyComment struct {
	ReplyID    string         `json:"reply_id"`
	Content    string         `json:"content"`
	Author     rawTikTokAuthor `json:"author"`
	LikesCount int            `json:"likes_count"`
	RepliedAt  string         `json:"replied_at"`
}

type rawTikTokAuthor struct {
	UID      string `json:"uid"`
	Username string `json:"username"`
	Nickname string `json:"nickname"`
	Avatar   string `json:"avatar"`
}

func validateParseAndStoreRawBatchInput(input uap.ParseAndStoreRawBatchInput) error {
	if strings.TrimSpace(input.RawBatchID) == "" ||
		strings.TrimSpace(input.ProjectID) == "" ||
		strings.TrimSpace(input.SourceID) == "" ||
		strings.TrimSpace(input.TaskID) == "" ||
		strings.TrimSpace(input.StorageBucket) == "" ||
		strings.TrimSpace(input.StoragePath) == "" ||
		strings.TrimSpace(input.BatchID) == "" {
		return uap.ErrInvalidRawBatchInput
	}

	if strings.ToLower(strings.TrimSpace(input.Platform)) != tikTokPlatform ||
		strings.TrimSpace(input.Action) != tikTokFullFlow {
		return uap.ErrInvalidRawBatchInput
	}

	return nil
}

func flattenTikTokFullFlow(rawBytes []byte, input uap.ParseAndStoreRawBatchInput, onRecord func(uap.UAPRecord)) ([]uap.UAPRecord, error) {
	var envelope rawTikTokFullFlowEnvelope
	if err := json.Unmarshal(rawBytes, &envelope); err != nil {
		return nil, uap.ErrParseRawPayload
	}

	records := make([]uap.UAPRecord, 0)
	for _, bundle := range envelope.Result.Posts {
		postRecord, rootID := mapTikTokPost(bundle, input)
		if rootID == "" {
			continue
		}
		records = append(records, postRecord)
		if onRecord != nil {
			onRecord(postRecord)
		}

		for _, comment := range bundle.Comments.Comments {
			commentRecord, commentID := mapTikTokComment(comment, input, rootID)
			if commentID == "" {
				continue
			}
			records = append(records, commentRecord)
			if onRecord != nil {
				onRecord(commentRecord)
			}

			for _, reply := range comment.ReplyComments {
				replyRecord := mapTikTokReply(reply, input, rootID, commentID)
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

func mapTikTokPost(bundle rawTikTokFullFlowPostBundle, input uap.ParseAndStoreRawBatchInput) (uap.UAPRecord, string) {
	videoID := strings.TrimSpace(firstNonEmpty(bundle.Detail.VideoID, bundle.Post.VideoID))
	if videoID == "" {
		return uap.UAPRecord{}, ""
	}

	rootID := buildUAPID("tt_p_", videoID)
	isVerified := false
	isShopVideo := bundle.Detail.IsShopVideo || bundle.Post.IsShopVideo
	postedAt := strings.TrimSpace(firstNonEmpty(bundle.Detail.PostedAt, bundle.Post.PostedAt))
	ingestedAt := input.CompletionTime.UTC().Format(timeRFC3339)

	return uap.UAPRecord{
		Identity: uap.UAPIdentity{
			UAPID:     rootID,
			OriginID:  videoID,
			UAPType:   uapTypePost,
			Platform:  tikTokPlatform,
			URL:       firstNonEmpty(bundle.Detail.URL, bundle.Post.URL),
			TaskID:    input.TaskID,
			ProjectID: input.ProjectID,
		},
		Hierarchy: uap.UAPHierarchy{
			ParentID: nil,
			RootID:   rootID,
			Depth:    0,
		},
		Content: uap.UAPContent{
			Text:           firstNonEmpty(bundle.Detail.Description, bundle.Post.Description),
			Hashtags:       normalizeStringSlice(firstNonEmptySlice(bundle.Detail.Hashtags, bundle.Post.Hashtags)),
			TikTokKeywords: normalizeStringSlice(bundle.Detail.Summary.Keywords),
			IsShopVideo:    &isShopVideo,
			MusicTitle:     bundle.Detail.MusicTitle,
			MusicURL:       firstNonEmpty(bundle.Detail.Downloads.Music, bundle.Detail.MusicURL),
			SummaryTitle:   bundle.Detail.Summary.Title,
			SubtitleURL:    firstNonEmpty(bundle.Detail.Downloads.Subtitle, bundle.Detail.SubtitleURL),
			Language:       strings.TrimSpace(bundle.Detail.Summary.Language),
		},
		Author: uap.UAPAuthor{
			ID:         firstNonEmpty(bundle.Detail.Author.UID, bundle.Post.Author.UID),
			Username:   firstNonEmpty(bundle.Detail.Author.Username, bundle.Post.Author.Username),
			Nickname:   firstNonEmpty(bundle.Detail.Author.Nickname, bundle.Post.Author.Nickname),
			Avatar:     firstNonEmpty(bundle.Detail.Author.Avatar, bundle.Post.Author.Avatar),
			IsVerified: &isVerified,
		},
		Engagement: uap.UAPEngagement{
			Likes:         intPtr(firstNonZero(bundle.Detail.LikesCount, bundle.Post.LikesCount)),
			CommentsCount: intPtr(firstNonZero(bundle.Detail.CommentsCount, bundle.Post.CommentsCount)),
			Shares:        intPtr(firstNonZero(bundle.Detail.SharesCount, bundle.Post.SharesCount)),
			Views:         intPtr(firstNonZero(bundle.Detail.ViewsCount, bundle.Post.ViewsCount)),
			Bookmarks:     intPtr(bundle.Detail.BookmarksCount),
		},
		Media: []uap.UAPMedia{
			{
				Type:        "video",
				URL:         firstNonEmpty(bundle.Detail.PlayURL, bundle.Post.URL),
				DownloadURL: firstNonEmpty(bundle.Detail.Downloads.Video, bundle.Detail.DownloadURL),
				Duration:    intPtr(bundle.Detail.Duration),
				Thumbnail:   firstNonEmpty(bundle.Detail.Downloads.Cover, bundle.Detail.CoverURL, bundle.Detail.OriginCoverURL),
			},
		},
		Temporal: uap.UAPTemporal{
			PostedAt:   postedAt,
			IngestedAt: ingestedAt,
		},
	}, rootID
}

func mapTikTokComment(comment rawTikTokComment, input uap.ParseAndStoreRawBatchInput, rootID string) (uap.UAPRecord, string) {
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
			UAPID:     buildUAPID("tt_c_", commentID),
			OriginID:  commentID,
			UAPType:   uapTypeComment,
			Platform:  tikTokPlatform,
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
			Likes:      intPtr(comment.LikesCount),
			ReplyCount: intPtr(comment.ReplyCount),
			SortScore:  float64Ptr(sortScore),
		},
		Temporal: uap.UAPTemporal{
			PostedAt: strings.TrimSpace(comment.CommentedAt),
		},
	}, commentID
}

func mapTikTokReply(reply rawTikTokReplyComment, input uap.ParseAndStoreRawBatchInput, rootID, commentID string) uap.UAPRecord {
	replyID := strings.TrimSpace(reply.ReplyID)
	if replyID == "" {
		return uap.UAPRecord{}
	}

	parentID := buildUAPID("tt_c_", commentID)

	return uap.UAPRecord{
		Identity: uap.UAPIdentity{
			UAPID:     buildUAPID("tt_r_", replyID),
			OriginID:  replyID,
			UAPType:   uapTypeReply,
			Platform:  tikTokPlatform,
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
			Likes: intPtr(reply.LikesCount),
		},
		Temporal: uap.UAPTemporal{
			PostedAt: strings.TrimSpace(reply.RepliedAt),
		},
	}
}

func chunkRecords(records []uap.UAPRecord) [][]uap.UAPRecord {
	if len(records) == 0 {
		return nil
	}

	chunks := make([][]uap.UAPRecord, 0, (len(records)+uapChunkSize-1)/uapChunkSize)
	for start := 0; start < len(records); start += uapChunkSize {
		end := start + uapChunkSize
		if end > len(records) {
			end = len(records)
		}
		chunks = append(chunks, records[start:end])
	}

	return chunks
}

func marshalChunkJSONL(records []uap.UAPRecord) ([]byte, error) {
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	for _, record := range records {
		if err := encoder.Encode(record); err != nil {
			return nil, err
		}
	}
	return buf.Bytes(), nil
}

func buildPartPath(projectID, sourceID, batchID string, partNo int) string {
	return path.Join(
		"uap-batches",
		strings.TrimSpace(projectID),
		strings.TrimSpace(sourceID),
		strings.TrimSpace(batchID),
		fmt.Sprintf("part-%05d.jsonl", partNo),
	)
}

func uploadChunk(ctx context.Context, client minioPkg.MinIO, bucket, projectID, sourceID, batchID string, partNo int, records []uap.UAPRecord) (artifactPart, error) {
	payload, err := marshalChunkJSONL(records)
	if err != nil {
		return artifactPart{}, err
	}

	objectPath := buildPartPath(projectID, sourceID, batchID, partNo)
	reader := bytes.NewReader(payload)
	if _, err := client.UploadFile(ctx, &minioPkg.UploadRequest{
		BucketName:   bucket,
		ObjectName:   objectPath,
		OriginalName: path.Base(objectPath),
		Reader:       reader,
		Size:         int64(len(payload)),
		ContentType:  uapContentType,
		Metadata: map[string]string{
			"project-id": projectID,
			"source-id":  sourceID,
			"batch-id":   batchID,
			"part-no":    strconv.Itoa(partNo),
		},
	}); err != nil {
		return artifactPart{}, err
	}

	return artifactPart{
		PartNo:        partNo,
		StorageBucket: bucket,
		StoragePath:   objectPath,
		RecordCount:   len(records),
	}, nil
}

func mergeRawMetadata(existing json.RawMessage, parts []artifactPart, totalRecords int, publishStats *kafkaPublishStats) (json.RawMessage, error) {
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

	root[uapArtifactsKey] = map[string]interface{}{
		"version":       uapArtifactsVersion,
		"chunk_size":    uapChunkSize,
		"total_records": totalRecords,
		"total_parts":   len(parts),
		"content_type":  uapContentType,
		"parts":         partPayload,
	}

	if publishStats != nil {
		root[uapKafkaPublishKey] = map[string]interface{}{
			"topic":           strings.TrimSpace(publishStats.Topic),
			"attempted_count": publishStats.AttemptedCount,
			"success_count":   publishStats.SuccessCount,
			"failed_count":    publishStats.FailedCount,
			"last_error":      strings.TrimSpace(publishStats.LastError),
		}
	}

	return json.Marshal(root)
}

func failRawBatch(
	ctx context.Context,
	uc *implUseCase,
	input uap.ParseAndStoreRawBatchInput,
	errMessage, publishErr string,
	parts []artifactPart,
	totalRecords int,
	publishStats *kafkaPublishStats,
) error {
	metadata, _ := mergeRawMetadata(input.RawMetadata, parts, totalRecords, publishStats)
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

func normalizeStringSlice(values []string) []string {
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

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func firstNonEmptySlice(values ...[]string) []string {
	for _, value := range values {
		if normalized := normalizeStringSlice(value); len(normalized) > 0 {
			return normalized
		}
	}
	return nil
}

func firstNonZero(values ...int) int {
	for _, value := range values {
		if value != 0 {
			return value
		}
	}
	return 0
}

func buildUAPID(prefix, originID string) string {
	return prefix + strings.TrimSpace(originID)
}

func intPtr(v int) *int {
	return &v
}

func float64Ptr(v float64) *float64 {
	return &v
}

const timeRFC3339 = "2006-01-02T15:04:05Z07:00"

func readAllAndClose(reader io.ReadCloser) ([]byte, error) {
	defer reader.Close()
	return io.ReadAll(reader)
}
