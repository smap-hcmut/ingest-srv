package usecase

import (
	"encoding/json"
	"strings"
	"time"

	"ingest-srv/internal/uap"
)

func (uc *implUseCase) flattenTikTokFullFlow(rawBytes []byte, input uap.ParseAndStoreRawBatchInput, onRecord func(uap.UAPRecord)) ([]uap.UAPRecord, error) {
	flattenInput, err := uc.parseTikTokFullFlowInput(rawBytes)
	if err != nil {
		return nil, err
	}

	crawlKeyword := uc.extractCrawlKeyword(input.RequestPayload)
	records := make([]uap.UAPRecord, 0)
	for _, bundle := range flattenInput.Posts {
		postRecord, rootID := uc.mapTikTokPost(bundle, input)
		if rootID == "" {
			continue
		}
		postRecord.CrawlKeyword = crawlKeyword
		records = append(records, postRecord)
		if onRecord != nil {
			onRecord(postRecord)
		}

		for _, comment := range bundle.Comments {
			commentRecord, commentID := uc.mapTikTokComment(comment, input, rootID)
			if commentID == "" {
				continue
			}
			commentRecord.CrawlKeyword = crawlKeyword
			records = append(records, commentRecord)
			if onRecord != nil {
				onRecord(commentRecord)
			}

			for _, reply := range comment.ReplyComments {
				replyRecord := uc.mapTikTokReply(reply, input, rootID, commentID)
				if replyRecord.Identity.OriginID == "" {
					continue
				}
				replyRecord.CrawlKeyword = crawlKeyword
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
	platformMeta := map[string]interface{}{
		"tiktok": map[string]interface{}{
			"music_title":   bundle.Detail.MusicTitle,
			"music_url":     uc.firstNonEmpty(bundle.Detail.Downloads.Music, bundle.Detail.MusicURL),
			"is_shop_video": isShopVideo,
		},
	}

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
			Text:     uc.firstNonEmpty(bundle.Detail.Description, bundle.Post.Description),
			Title:    strings.TrimSpace(bundle.Detail.Summary.Title),
			Subtitle: uc.resolveTikTokSubtitleText(bundle.Detail),
			Hashtags: uc.normalizeStringSlice(uc.firstNonEmptySlice(bundle.Detail.Hashtags, bundle.Post.Hashtags)),
			Keywords: uc.normalizeStringSlice(bundle.Detail.Summary.Keywords),
			Language: strings.TrimSpace(bundle.Detail.Summary.Language),
		},
		Author: uap.UAPAuthor{
			ID:         uc.firstNonEmpty(bundle.Detail.Author.UID, bundle.Post.Author.UID),
			Username:   uc.firstNonEmpty(bundle.Detail.Author.Username, bundle.Post.Author.Username),
			Nickname:   uc.firstNonEmpty(bundle.Detail.Author.Nickname, bundle.Post.Author.Nickname),
			Avatar:     uc.firstNonEmpty(bundle.Detail.Author.Avatar, bundle.Post.Author.Avatar),
			ProfileURL: "",
			IsVerified: &isVerified,
		},
		Engagement: uap.UAPEngagement{
			Likes:         uc.intPtr(uc.firstNonZero(bundle.Detail.LikesCount, bundle.Post.LikesCount)),
			CommentsCount: uc.intPtr(uc.firstNonZero(bundle.Detail.CommentsCount, bundle.Post.CommentsCount)),
			Shares:        uc.intPtr(uc.firstNonZero(bundle.Detail.SharesCount, bundle.Post.SharesCount)),
			Views:         uc.intPtr(uc.firstNonZero(bundle.Detail.ViewsCount, bundle.Post.ViewsCount)),
			Saves:         uc.intPtr(bundle.Detail.BookmarksCount),
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
		PlatformMeta: platformMeta,
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
	platformMeta := map[string]interface{}{
		"tiktok": map[string]interface{}{
			"sort_score": sortScore,
		},
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
			ID:         comment.Author.UID,
			Username:   comment.Author.Username,
			Nickname:   comment.Author.Nickname,
			Avatar:     comment.Author.Avatar,
			ProfileURL: "",
		},
		Engagement: uap.UAPEngagement{
			Likes:      uc.intPtr(comment.LikesCount),
			ReplyCount: uc.intPtr(comment.ReplyCount),
		},
		Temporal: uap.UAPTemporal{
			PostedAt: strings.TrimSpace(comment.CommentedAt),
		},
		PlatformMeta: platformMeta,
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
			ID:         reply.Author.UID,
			Username:   reply.Author.Username,
			Nickname:   reply.Author.Nickname,
			Avatar:     reply.Author.Avatar,
			ProfileURL: "",
		},
		Engagement: uap.UAPEngagement{
			Likes: uc.intPtr(reply.LikesCount),
		},
		Temporal: uap.UAPTemporal{
			PostedAt: strings.TrimSpace(reply.RepliedAt),
		},
		PlatformMeta: map[string]interface{}{},
	}
}

func (uc *implUseCase) resolveTikTokSubtitleText(detail uap.TikTokDetailInput) string {
	return uc.resolveSubtitleText(
		detail.SubtitleURL,
		detail.Downloads.Subtitle,
	)
}
