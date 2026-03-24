package usecase

import (
	"encoding/json"
	"strings"
	"time"

	"ingest-srv/internal/uap"
)

func (uc *implUseCase) flattenYouTubeFullFlow(rawBytes []byte, input uap.ParseAndStoreRawBatchInput, onRecord func(uap.UAPRecord)) ([]uap.UAPRecord, error) {
	flattenInput, err := uc.parseYouTubeFullFlowInput(rawBytes)
	if err != nil {
		return nil, err
	}

	crawlKeyword := uc.extractCrawlKeyword(input.RequestPayload)
	records := make([]uap.UAPRecord, 0)
	for _, bundle := range flattenInput.Videos {
		postRecord, rootID := uc.mapYouTubePost(bundle, input)
		if rootID == "" {
			continue
		}
		postRecord.CrawlKeyword = crawlKeyword
		records = append(records, postRecord)
		if onRecord != nil {
			onRecord(postRecord)
		}

		for _, comment := range bundle.Comments.Comments {
			commentRecord := uc.mapYouTubeComment(comment, input, rootID)
			if strings.TrimSpace(commentRecord.Identity.OriginID) == "" {
				continue
			}
			commentRecord.CrawlKeyword = crawlKeyword
			records = append(records, commentRecord)
			if onRecord != nil {
				onRecord(commentRecord)
			}
		}
	}

	return records, nil
}

func (uc *implUseCase) parseYouTubeFullFlowInput(rawBytes []byte) (uap.YouTubeFullFlowInput, error) {
	var root map[string]interface{}
	if err := json.Unmarshal(rawBytes, &root); err != nil {
		return uap.YouTubeFullFlowInput{}, uap.ErrParseRawPayload
	}

	result := root
	if candidate := uc.toMap(root["result"]); len(candidate) > 0 {
		result = candidate
	}

	videoItems := uc.toSlice(result["videos"])
	videos := make([]uap.YouTubeVideoBundleInput, 0, len(videoItems))
	for _, item := range videoItems {
		bundleMap := uc.toMap(item)
		videoMap := uc.toMap(bundleMap["video"])
		if len(videoMap) == 0 {
			continue
		}

		detailMap := uc.toMap(bundleMap["detail"])
		commentsMap := uc.toMap(bundleMap["comments"])
		transcriptMap := uc.toMap(bundleMap["transcript"])

		commentItems := uc.toSlice(commentsMap["comments"])
		comments := make([]uap.YouTubeCommentInput, 0, len(commentItems))
		for _, commentItem := range commentItems {
			commentMap := uc.toMap(commentItem)
			comments = append(comments, uap.YouTubeCommentInput{
				CommentID:          uc.stringAt(commentMap, "comment_id"),
				VideoID:            uc.stringAt(commentMap, "video_id"),
				AuthorName:         uc.stringAt(commentMap, "author_name"),
				AuthorChannelID:    uc.stringAt(commentMap, "author_channel_id"),
				AuthorThumbnailURL: uc.stringAt(commentMap, "author_thumbnail_url"),
				Content:            uc.stringAt(commentMap, "content"),
				LikesCount:         uc.intAt(commentMap, "likes_count"),
				ReplyCount:         uc.intAt(commentMap, "reply_count"),
				PublishedTimeText:  uc.stringAt(commentMap, "published_time_text"),
			})
		}

		segmentItems := uc.toSlice(transcriptMap["segments"])
		segments := make([]uap.YouTubeTranscriptSegmentInput, 0, len(segmentItems))
		for _, segmentItem := range segmentItems {
			segmentMap := uc.toMap(segmentItem)
			segments = append(segments, uap.YouTubeTranscriptSegmentInput{
				StartMS:       uc.intAt(segmentMap, "start_ms"),
				EndMS:         uc.intAt(segmentMap, "end_ms"),
				Text:          uc.stringAt(segmentMap, "text"),
				StartTimeText: uc.stringAt(segmentMap, "start_time_text"),
			})
		}

		videos = append(videos, uap.YouTubeVideoBundleInput{
			Video: uap.YouTubeVideoInput{
				VideoID:            uc.stringAt(videoMap, "video_id"),
				Title:              uc.stringAt(videoMap, "title"),
				ChannelName:        uc.stringAt(videoMap, "channel_name"),
				ChannelID:          uc.stringAt(videoMap, "channel_id"),
				ViewsCount:         uc.intAt(videoMap, "views_count"),
				ViewsText:          uc.stringAt(videoMap, "views_text"),
				DurationText:       uc.stringAt(videoMap, "duration_text"),
				PublishedTimeText:  uc.stringAt(videoMap, "published_time_text"),
				ThumbnailURL:       uc.stringAt(videoMap, "thumbnail_url"),
				DescriptionSnippet: uc.stringAt(videoMap, "description_snippet"),
				URL:                uc.stringAt(videoMap, "url"),
			},
			Detail: uap.YouTubeDetailInput{
				VideoID:       uc.stringAt(detailMap, "video_id"),
				Title:         uc.stringAt(detailMap, "title"),
				Description:   uc.stringAt(detailMap, "description"),
				Keywords:      uc.stringSliceAt(detailMap, "keywords"),
				Width:         uc.intAt(detailMap, "width"),
				Height:        uc.intAt(detailMap, "height"),
				AuthorName:    uc.stringAt(detailMap, "author_name"),
				AuthorURL:     uc.stringAt(detailMap, "author_url"),
				LikesCount:    uc.intAt(detailMap, "likes_count"),
				ViewsCount:    uc.intAt(detailMap, "views_count"),
				DatePublished: uc.stringAt(detailMap, "date_published"),
				UploadDate:    uc.stringAt(detailMap, "upload_date"),
			},
			Comments: uap.YouTubeCommentsInput{
				VideoID:  uc.stringAt(commentsMap, "video_id"),
				Comments: comments,
				Total:    uc.intAt(commentsMap, "total"),
			},
			Transcript: uap.YouTubeTranscriptInput{
				FullText: uc.stringAt(transcriptMap, "full_text"),
				Segments: segments,
			},
		})
	}

	return uap.YouTubeFullFlowInput{Videos: videos}, nil
}

func (uc *implUseCase) mapYouTubePost(bundle uap.YouTubeVideoBundleInput, input uap.ParseAndStoreRawBatchInput) (uap.UAPRecord, string) {
	videoID := strings.TrimSpace(bundle.Video.VideoID)
	if videoID == "" {
		return uap.UAPRecord{}, ""
	}

	rootID := uc.buildUAPID("yt_p_", videoID)
	ingestedAt := input.CompletionTime.UTC().Format(time.RFC3339)
	subtitle := uc.firstNonEmpty(
		strings.TrimSpace(bundle.Transcript.FullText),
		uc.joinTranscriptSegments(bundle.Transcript.Segments),
	)
	profileURL := uc.firstNonEmpty(bundle.Detail.AuthorURL, uc.buildYouTubeChannelURL(bundle.Video.ChannelID))
	platformMeta := map[string]interface{}{
		"youtube": map[string]interface{}{
			"published_time_text": bundle.Video.PublishedTimeText,
			"views_text":          bundle.Video.ViewsText,
			"description_snippet": bundle.Video.DescriptionSnippet,
		},
	}

	return uap.UAPRecord{
		Identity: uap.UAPIdentity{
			UAPID:     rootID,
			OriginID:  videoID,
			UAPType:   uap.UAPTypePost,
			Platform:  uap.PlatformYouTube,
			URL:       strings.TrimSpace(bundle.Video.URL),
			TaskID:    input.TaskID,
			ProjectID: input.ProjectID,
		},
		Hierarchy: uap.UAPHierarchy{
			ParentID: nil,
			RootID:   rootID,
			Depth:    0,
		},
		Content: uap.UAPContent{
			Text:     uc.firstNonEmpty(bundle.Detail.Description, bundle.Video.DescriptionSnippet),
			Title:    uc.firstNonEmpty(bundle.Detail.Title, bundle.Video.Title),
			Subtitle: subtitle,
			Keywords: uc.normalizeStringSlice(bundle.Detail.Keywords),
			Links:    uc.extractLinks(bundle.Detail.Description, bundle.Video.DescriptionSnippet, subtitle),
		},
		Author: uap.UAPAuthor{
			ID:         strings.TrimSpace(bundle.Video.ChannelID),
			Username:   uc.extractYouTubeHandle(bundle.Detail.AuthorURL),
			Nickname:   uc.firstNonEmpty(bundle.Detail.AuthorName, bundle.Video.ChannelName),
			Avatar:     "",
			ProfileURL: profileURL,
		},
		Engagement: uap.UAPEngagement{
			Likes:         uc.intPtr(bundle.Detail.LikesCount),
			CommentsCount: uc.intPtr(bundle.Comments.Total),
			Views:         uc.intPtr(uc.firstNonZero(bundle.Detail.ViewsCount, bundle.Video.ViewsCount)),
		},
		Media: []uap.UAPMedia{
			{
				Type:      "video",
				URL:       strings.TrimSpace(bundle.Video.URL),
				Thumbnail: strings.TrimSpace(bundle.Video.ThumbnailURL),
				Duration:  uc.parseDurationText(bundle.Video.DurationText),
				Width:     uc.intPtr(bundle.Detail.Width),
				Height:    uc.intPtr(bundle.Detail.Height),
			},
		},
		Temporal: uap.UAPTemporal{
			PostedAt:   uc.firstNonEmpty(bundle.Detail.DatePublished, bundle.Detail.UploadDate),
			UpdatedAt:  "",
			IngestedAt: ingestedAt,
		},
		PlatformMeta: platformMeta,
	}, rootID
}

func (uc *implUseCase) mapYouTubeComment(comment uap.YouTubeCommentInput, input uap.ParseAndStoreRawBatchInput, rootID string) uap.UAPRecord {
	commentID := strings.TrimSpace(comment.CommentID)
	if commentID == "" {
		return uap.UAPRecord{}
	}

	parentID := rootID
	profileURL := uc.buildYouTubeChannelURL(comment.AuthorChannelID)

	return uap.UAPRecord{
		Identity: uap.UAPIdentity{
			UAPID:     uc.buildUAPID("yt_c_", commentID),
			OriginID:  commentID,
			UAPType:   uap.UAPTypeComment,
			Platform:  uap.PlatformYouTube,
			URL:       "",
			TaskID:    input.TaskID,
			ProjectID: input.ProjectID,
		},
		Hierarchy: uap.UAPHierarchy{
			ParentID: &parentID,
			RootID:   rootID,
			Depth:    1,
		},
		Content: uap.UAPContent{
			Text:  strings.TrimSpace(comment.Content),
			Links: uc.extractLinks(comment.Content),
		},
		Author: uap.UAPAuthor{
			ID:         strings.TrimSpace(comment.AuthorChannelID),
			Username:   strings.TrimSpace(comment.AuthorName),
			Nickname:   strings.TrimSpace(comment.AuthorName),
			Avatar:     strings.TrimSpace(comment.AuthorThumbnailURL),
			ProfileURL: profileURL,
		},
		Engagement: uap.UAPEngagement{
			Likes:      uc.intPtr(comment.LikesCount),
			ReplyCount: uc.intPtr(comment.ReplyCount),
		},
		Temporal: uap.UAPTemporal{
			PostedAt:   "",
			UpdatedAt:  "",
			IngestedAt: input.CompletionTime.UTC().Format(time.RFC3339),
		},
		PlatformMeta: map[string]interface{}{
			"youtube": map[string]interface{}{
				"published_time_text": comment.PublishedTimeText,
			},
		},
	}
}
