package uap

import (
	"encoding/json"
	"time"
)

type ParseAndStoreRawBatchInput struct {
	RawBatchID     string
	ProjectID      string
	SourceID       string
	ExternalTaskID string
	TaskID         string
	Platform       string
	Action         string
	StorageBucket  string
	StoragePath    string
	BatchID        string
	RawMetadata    json.RawMessage
	RequestPayload json.RawMessage
	CompletionTime time.Time
}

type TikTokFullFlowInput struct {
	Posts []TikTokPostBundleInput
}

type TikTokPostBundleInput struct {
	Post     TikTokPostInput
	Detail   TikTokDetailInput
	Comments []TikTokCommentInput
}

type TikTokPostInput struct {
	VideoID       string
	URL           string
	Description   string
	Author        TikTokAuthorInput
	LikesCount    int
	CommentsCount int
	SharesCount   int
	ViewsCount    int
	Hashtags      []string
	PostedAt      string
	IsShopVideo   bool
}

type TikTokDetailInput struct {
	VideoID        string
	URL            string
	Description    string
	Author         TikTokAuthorInput
	LikesCount     int
	CommentsCount  int
	SharesCount    int
	ViewsCount     int
	BookmarksCount int
	Hashtags       []string
	MusicTitle     string
	MusicURL       string
	Duration       int
	PostedAt       string
	IsShopVideo    bool
	Summary        TikTokSummaryInput
	PlayURL        string
	DownloadURL    string
	CoverURL       string
	OriginCoverURL string
	SubtitleURL    string
	Downloads      TikTokDetailAssetsInput
}

type TikTokSummaryInput struct {
	Title    string
	Keywords []string
	Language string
}

type TikTokDetailAssetsInput struct {
	Music    string
	Cover    string
	Subtitle string
	Video    string
}

type TikTokCommentInput struct {
	CommentID      string
	Content        string
	Author         TikTokAuthorInput
	LikesCount     int
	ReplyCount     int
	SortExtraScore TikTokCommentScoreInput
	ReplyComments  []TikTokReplyInput
	CommentedAt    string
}

type TikTokCommentScoreInput struct {
	ReplyScore    float64
	ShowMoreScore float64
}

type TikTokReplyInput struct {
	ReplyID    string
	Content    string
	Author     TikTokAuthorInput
	LikesCount int
	RepliedAt  string
}

type TikTokAuthorInput struct {
	UID      string
	Username string
	Nickname string
	Avatar   string
}

type YouTubeFullFlowInput struct {
	Videos []YouTubeVideoBundleInput
}

type YouTubeVideoBundleInput struct {
	Video      YouTubeVideoInput
	Detail     YouTubeDetailInput
	Comments   YouTubeCommentsInput
	Transcript YouTubeTranscriptInput
}

type YouTubeVideoInput struct {
	VideoID            string
	Title              string
	ChannelName        string
	ChannelID          string
	ViewsCount         int
	ViewsText          string
	DurationText       string
	PublishedTimeText  string
	ThumbnailURL       string
	DescriptionSnippet string
	URL                string
}

type YouTubeDetailInput struct {
	VideoID       string
	Title         string
	Description   string
	Keywords      []string
	Width         int
	Height        int
	AuthorName    string
	AuthorURL     string
	LikesCount    int
	ViewsCount    int
	DatePublished string
	UploadDate    string
}

type YouTubeCommentsInput struct {
	VideoID  string
	Comments []YouTubeCommentInput
	Total    int
}

type YouTubeCommentInput struct {
	CommentID          string
	VideoID            string
	AuthorName         string
	AuthorChannelID    string
	AuthorThumbnailURL string
	Content            string
	LikesCount         int
	ReplyCount         int
	PublishedTimeText  string
}

type YouTubeTranscriptInput struct {
	FullText string
	Segments []YouTubeTranscriptSegmentInput
}

type YouTubeTranscriptSegmentInput struct {
	StartMS       int
	EndMS         int
	Text          string
	StartTimeText string
}

type FacebookFullFlowInput struct {
	Posts []FacebookPostBundleInput
}

type FacebookPostBundleInput struct {
	Post     FacebookPostInput
	Comments FacebookCommentsInput
}

type FacebookPostInput struct {
	PostID        string
	Message       string
	URL           string
	Author        FacebookPostAuthorInput
	CreatedTime   int64
	ReactionCount int
	CommentCount  int
	ShareCount    int
	Attachments   []FacebookAttachmentInput
}

type FacebookPostAuthorInput struct {
	ID        string
	Name      string
	URL       string
	AvatarURL string
}

type FacebookAttachmentInput struct {
	Type        string
	URL         string
	MediaURL    string
	Width       int
	Height      int
	Title       string
	Description string
}

type FacebookCommentsInput struct {
	PostID   string
	Total    int
	Comments []FacebookCommentInput
}

type FacebookCommentInput struct {
	ID            string
	Message       string
	Author        FacebookCommentAuthorInput
	CreatedTime   int64
	ReactionCount int
	ReplyCount    int
	Replies       []FacebookCommentInput
}

type FacebookCommentAuthorInput struct {
	ID         string
	Name       string
	ProfileURL string
	AvatarURL  string
}

type PublishUAPInput struct {
	Record UAPRecord
}

type UAPType string

const (
	UAPTypePost    UAPType = "POST"
	UAPTypeComment UAPType = "COMMENT"
	UAPTypeReply   UAPType = "REPLY"
)

const (
	PlatformTikTok   = "tiktok"
	PlatformYouTube  = "youtube"
	PlatformFacebook = "facebook"
	TaskTypeFullFlow = "full_flow"
)

const (
	ChunkSize            = 20
	ContentTypeNDJSON    = "application/x-ndjson"
	ArtifactsMetadataKey = "uap_artifacts"
	ArtifactsVersionV1   = "v1"
	KafkaPublishKey      = "kafka_publish"
)

// ArtifactPart describes one UAP artifact chunk written to object storage.
type ArtifactPart struct {
	PartNo        int
	StorageBucket string
	StoragePath   string
	RecordCount   int
}

// KafkaPublishStats aggregates one raw-batch publish attempt summary.
type KafkaPublishStats struct {
	Topic          string
	AttemptedCount int
	SuccessCount   int
	FailedCount    int
	LastError      string
}

type UAPRecord struct {
	Identity     UAPIdentity
	Hierarchy    UAPHierarchy
	Content      UAPContent
	Author       UAPAuthor
	Engagement   UAPEngagement
	Media        []UAPMedia
	Temporal     UAPTemporal
	PlatformMeta map[string]interface{}
}

type UAPIdentity struct {
	UAPID     string
	OriginID  string
	UAPType   UAPType
	Platform  string
	URL       string
	TaskID    string
	ProjectID string
}

type UAPHierarchy struct {
	ParentID *string
	RootID   string
	Depth    int
}

type UAPContent struct {
	Text     string
	Title    string
	Subtitle string
	Hashtags []string
	Keywords []string
	Language string
	Links    []string
}

type UAPAuthor struct {
	ID         string
	Username   string
	Nickname   string
	Avatar     string
	ProfileURL string
	IsVerified *bool
}

type UAPEngagement struct {
	Likes         *int
	CommentsCount *int
	Shares        *int
	Views         *int
	Saves         *int
	ReplyCount    *int
}

type UAPMedia struct {
	Type        string
	URL         string
	DownloadURL string
	Duration    *int
	Thumbnail   string
	Width       *int
	Height      *int
}

type UAPTemporal struct {
	PostedAt   string
	UpdatedAt  string
	IngestedAt string
}
