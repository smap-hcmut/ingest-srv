package usecase

import (
	"encoding/json"
	"strings"
	"time"

	"ingest-srv/internal/uap"
)

func (uc *implUseCase) flattenFacebookFullFlow(rawBytes []byte, input uap.ParseAndStoreRawBatchInput, onRecord func(uap.UAPRecord)) ([]uap.UAPRecord, error) {
	flattenInput, err := uc.parseFacebookFullFlowInput(rawBytes)
	if err != nil {
		return nil, err
	}

	records := make([]uap.UAPRecord, 0)
	for _, bundle := range flattenInput.Posts {
		postRecord, rootID := uc.mapFacebookPost(bundle, input)
		if rootID == "" {
			continue
		}
		records = append(records, postRecord)
		if onRecord != nil {
			onRecord(postRecord)
		}

		for _, comment := range bundle.Comments.Comments {
			commentRecord := uc.mapFacebookComment(comment, input, rootID)
			if strings.TrimSpace(commentRecord.Identity.OriginID) == "" {
				continue
			}
			records = append(records, commentRecord)
			if onRecord != nil {
				onRecord(commentRecord)
			}
		}
	}

	return records, nil
}

func (uc *implUseCase) parseFacebookFullFlowInput(rawBytes []byte) (uap.FacebookFullFlowInput, error) {
	var root map[string]interface{}
	if err := json.Unmarshal(rawBytes, &root); err != nil {
		return uap.FacebookFullFlowInput{}, uap.ErrParseRawPayload
	}

	result := root
	if candidate := uc.toMap(root["result"]); len(candidate) > 0 {
		result = candidate
	}

	postItems := uc.toSlice(result["posts"])
	posts := make([]uap.FacebookPostBundleInput, 0, len(postItems))
	for _, item := range postItems {
		bundleMap := uc.toMap(item)
		postMap := uc.toMap(bundleMap["post"])
		if len(postMap) == 0 {
			continue
		}

		commentsMap := uc.toMap(bundleMap["comments"])
		commentItems := uc.toSlice(commentsMap["comments"])
		comments := make([]uap.FacebookCommentInput, 0, len(commentItems))
		for _, commentItem := range commentItems {
			commentMap := uc.toMap(commentItem)
			authorMap := uc.toMap(commentMap["author"])
			replies := uc.toSlice(commentMap["replies"])
			comments = append(comments, uap.FacebookCommentInput{
				ID:      uc.stringAt(commentMap, "id"),
				Message: uc.stringAt(commentMap, "message"),
				Author: uap.FacebookCommentAuthorInput{
					ID:         uc.stringAt(authorMap, "id"),
					Name:       uc.stringAt(authorMap, "name"),
					ProfileURL: uc.stringAt(authorMap, "profile_url"),
					AvatarURL:  uc.stringAt(authorMap, "avatar_url"),
				},
				CreatedTime:   int64(uc.intAt(commentMap, "created_time")),
				ReactionCount: uc.intAt(commentMap, "reaction_count"),
				ReplyCount:    uc.intAt(commentMap, "reply_count"),
				Replies:       make([]uap.FacebookCommentInput, 0, len(replies)),
			})
		}

		authorMap := uc.toMap(postMap["author"])
		attachmentItems := uc.toSlice(postMap["attachments"])
		attachments := make([]uap.FacebookAttachmentInput, 0, len(attachmentItems))
		for _, attachmentItem := range attachmentItems {
			attachmentMap := uc.toMap(attachmentItem)
			attachments = append(attachments, uap.FacebookAttachmentInput{
				Type:        uc.stringAt(attachmentMap, "type"),
				URL:         uc.stringAt(attachmentMap, "url"),
				MediaURL:    uc.stringAt(attachmentMap, "media_url"),
				Width:       uc.intAt(attachmentMap, "width"),
				Height:      uc.intAt(attachmentMap, "height"),
				Title:       uc.stringAt(attachmentMap, "title"),
				Description: uc.stringAt(attachmentMap, "description"),
			})
		}

		posts = append(posts, uap.FacebookPostBundleInput{
			Post: uap.FacebookPostInput{
				PostID:  uc.stringAt(postMap, "post_id"),
				Message: uc.stringAt(postMap, "message"),
				URL:     uc.stringAt(postMap, "url"),
				Author: uap.FacebookPostAuthorInput{
					ID:        uc.stringAt(authorMap, "id"),
					Name:      uc.stringAt(authorMap, "name"),
					URL:       uc.stringAt(authorMap, "url"),
					AvatarURL: uc.stringAt(authorMap, "avatar_url"),
				},
				CreatedTime:   int64(uc.intAt(postMap, "created_time")),
				ReactionCount: uc.intAt(postMap, "reaction_count"),
				CommentCount:  uc.intAt(postMap, "comment_count"),
				ShareCount:    uc.intAt(postMap, "share_count"),
				Attachments:   attachments,
			},
			Comments: uap.FacebookCommentsInput{
				PostID:   uc.stringAt(commentsMap, "post_id"),
				Total:    uc.intAt(commentsMap, "total_count"),
				Comments: comments,
			},
		})
	}

	return uap.FacebookFullFlowInput{Posts: posts}, nil
}

func (uc *implUseCase) mapFacebookPost(bundle uap.FacebookPostBundleInput, input uap.ParseAndStoreRawBatchInput) (uap.UAPRecord, string) {
	postID := strings.TrimSpace(bundle.Post.PostID)
	if postID == "" {
		return uap.UAPRecord{}, ""
	}

	rootID := uc.buildUAPID("fb_p_", postID)

	return uap.UAPRecord{
		Identity: uap.UAPIdentity{
			UAPID:     rootID,
			OriginID:  postID,
			UAPType:   uap.UAPTypePost,
			Platform:  uap.PlatformFacebook,
			URL:       strings.TrimSpace(bundle.Post.URL),
			TaskID:    input.TaskID,
			ProjectID: input.ProjectID,
		},
		Hierarchy: uap.UAPHierarchy{
			ParentID: nil,
			RootID:   rootID,
			Depth:    0,
		},
		Content: uap.UAPContent{
			Text:  strings.TrimSpace(bundle.Post.Message),
			Links: uc.extractLinks(bundle.Post.Message),
		},
		Author: uap.UAPAuthor{
			ID:         strings.TrimSpace(bundle.Post.Author.ID),
			Username:   "",
			Nickname:   strings.TrimSpace(bundle.Post.Author.Name),
			Avatar:     strings.TrimSpace(bundle.Post.Author.AvatarURL),
			ProfileURL: strings.TrimSpace(bundle.Post.Author.URL),
		},
		Engagement: uap.UAPEngagement{
			Likes:         uc.intPtr(bundle.Post.ReactionCount),
			CommentsCount: uc.intPtr(bundle.Post.CommentCount),
			Shares:        uc.intPtr(bundle.Post.ShareCount),
		},
		Media: uc.mapFacebookAttachments(bundle.Post.Attachments),
		Temporal: uap.UAPTemporal{
			PostedAt:   uc.normalizeFacebookUnixTime(bundle.Post.CreatedTime),
			UpdatedAt:  "",
			IngestedAt: input.CompletionTime.UTC().Format(time.RFC3339),
		},
		PlatformMeta: map[string]interface{}{
			"facebook": map[string]interface{}{
				"attachment_count": len(bundle.Post.Attachments),
				"has_attachments":  len(bundle.Post.Attachments) > 0,
			},
		},
	}, rootID
}

func (uc *implUseCase) mapFacebookComment(comment uap.FacebookCommentInput, input uap.ParseAndStoreRawBatchInput, rootID string) uap.UAPRecord {
	commentID := strings.TrimSpace(comment.ID)
	if commentID == "" {
		return uap.UAPRecord{}
	}

	parentID := rootID
	repliesPresent := len(comment.Replies) > 0

	return uap.UAPRecord{
		Identity: uap.UAPIdentity{
			UAPID:     uc.buildUAPID("fb_c_", commentID),
			OriginID:  commentID,
			UAPType:   uap.UAPTypeComment,
			Platform:  uap.PlatformFacebook,
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
			Text:  strings.TrimSpace(comment.Message),
			Links: uc.extractLinks(comment.Message),
		},
		Author: uap.UAPAuthor{
			ID:         strings.TrimSpace(comment.Author.ID),
			Username:   "",
			Nickname:   strings.TrimSpace(comment.Author.Name),
			Avatar:     strings.TrimSpace(comment.Author.AvatarURL),
			ProfileURL: strings.TrimSpace(comment.Author.ProfileURL),
		},
		Engagement: uap.UAPEngagement{
			Likes:      uc.intPtr(comment.ReactionCount),
			ReplyCount: uc.intPtr(comment.ReplyCount),
		},
		Temporal: uap.UAPTemporal{
			PostedAt:   uc.normalizeFacebookUnixTime(comment.CreatedTime),
			UpdatedAt:  "",
			IngestedAt: input.CompletionTime.UTC().Format(time.RFC3339),
		},
		PlatformMeta: map[string]interface{}{
			"facebook": map[string]interface{}{
				"replies_present": repliesPresent,
			},
		},
	}
}

func (uc *implUseCase) mapFacebookAttachments(attachments []uap.FacebookAttachmentInput) []uap.UAPMedia {
	if len(attachments) == 0 {
		return nil
	}

	media := make([]uap.UAPMedia, 0, len(attachments))
	for _, attachment := range attachments {
		mediaType := ""
		switch strings.TrimSpace(strings.ToLower(attachment.Type)) {
		case "photo":
			mediaType = "image"
		case "video":
			mediaType = "video"
		default:
			continue
		}

		media = append(media, uap.UAPMedia{
			Type:        mediaType,
			URL:         strings.TrimSpace(attachment.URL),
			DownloadURL: strings.TrimSpace(attachment.MediaURL),
			Thumbnail:   "",
			Width:       uc.intPtrIfPositive(attachment.Width),
			Height:      uc.intPtrIfPositive(attachment.Height),
		})
	}

	if len(media) == 0 {
		return nil
	}
	return media
}

func (uc *implUseCase) normalizeFacebookUnixTime(ts int64) string {
	if ts <= 0 {
		return ""
	}
	return time.Unix(ts, 0).UTC().Format(time.RFC3339)
}
