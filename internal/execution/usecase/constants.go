package usecase

import executionRabbit "ingest-srv/internal/execution/delivery/rabbitmq"

const (
	tikTokTasksQueue         = executionRabbit.TikTokTasksQueueName
	facebookTasksQueue       = executionRabbit.FacebookTasksQueueName
	youtubeTasksQueue        = executionRabbit.YoutubeTasksQueueName
	minioVerifyRetryAttempts = 3
	defaultMinIntervalMinute = 1
	defaultMaxIntervalMinute = 1440
	normalModeMultiplier     = 1.0
	crisisModeMultiplier     = 0.2
	sleepModeMultiplier      = 5.0
	tikTokFullFlowLimit      = 50
	tikTokFullFlowThreshold  = 0.3
	tikTokFullFlowCommentCount = 500
)

const (
	actionSearch               = "search"
	actionPostDetail           = "post_detail"
	actionComments             = "comments"
	actionSummary              = "summary"
	actionCommentReplies       = "comment_replies"
	actionCookieCheck          = "cookie_check"
	actionFullFlow             = "full_flow"
	actionPosts                = "posts"
	actionCommentsGraphQL      = "comments_graphql"
	actionCommentsGraphQLBatch = "comments_graphql_batch"
	actionVideos               = "videos"
	actionVideoDetail          = "video_detail"
	actionTranscript           = "transcript"
)
