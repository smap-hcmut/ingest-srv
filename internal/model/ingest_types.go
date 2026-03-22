package model

type SourceType string

const (
	SourceTypeTikTok     SourceType = "TIKTOK"      // Crawl source lấy dữ liệu từ TikTok.
	SourceTypeFacebook   SourceType = "FACEBOOK"    // Crawl source lấy dữ liệu từ Facebook.
	SourceTypeYouTube    SourceType = "YOUTUBE"     // Crawl source lấy dữ liệu từ YouTube.
	SourceTypeFileUpload SourceType = "FILE_UPLOAD" // Passive source lấy dữ liệu do user upload.
	SourceTypeWebhook    SourceType = "WEBHOOK"     // Passive source nhận dữ liệu push từ hệ ngoài.
)

type SourceCategory string

const (
	SourceCategoryCrawl   SourceCategory = "CRAWL"   // Source chủ động đi lấy dữ liệu theo lịch.
	SourceCategoryPassive SourceCategory = "PASSIVE" // Source thụ động nhận dữ liệu từ file hoặc webhook.
)

type SourceStatus string

const (
	SourceStatusPending   SourceStatus = "PENDING"   // Mới tạo hoặc vừa sửa config, chưa sẵn sàng chạy.
	SourceStatusReady     SourceStatus = "READY"     // Đã dry run/validate xong, chờ project hoặc system activate.
	SourceStatusActive    SourceStatus = "ACTIVE"    // Đang chạy thật trong runtime.
	SourceStatusPaused    SourceStatus = "PAUSED"    // Tạm dừng nhưng còn khả năng resume.
	SourceStatusFailed    SourceStatus = "FAILED"    // Lỗi và cần can thiệp trước khi chạy lại.
	SourceStatusCompleted SourceStatus = "COMPLETED" // Chỉ dùng cho one-shot passive source như file upload.
	SourceStatusArchived  SourceStatus = "ARCHIVED"  // Ngừng vận hành và chỉ giữ lịch sử.
)

type OnboardingStatus string

const (
	OnboardingStatusNotRequired OnboardingStatus = "NOT_REQUIRED" // Source không cần bước onboarding/mapping.
	OnboardingStatusPending     OnboardingStatus = "PENDING"      // Đã vào flow onboarding nhưng chưa xử lý.
	OnboardingStatusAnalyzing   OnboardingStatus = "ANALYZING"    // Đang phân tích sample để gợi ý mapping.
	OnboardingStatusSuggested   OnboardingStatus = "SUGGESTED"    // Đã có mapping đề xuất, chờ confirm/chỉnh sửa.
	OnboardingStatusConfirmed   OnboardingStatus = "CONFIRMED"    // Mapping đã được chốt và có thể dùng runtime.
	OnboardingStatusFailed      OnboardingStatus = "FAILED"       // Onboarding lỗi, cần user/system sửa lại input.
)

type DryrunStatus string

const (
	DryrunStatusNotRequired DryrunStatus = "NOT_REQUIRED" // Source không cần dry run trước khi chạy.
	DryrunStatusPending     DryrunStatus = "PENDING"      // Đã tạo request dry run nhưng chưa bắt đầu.
	DryrunStatusRunning     DryrunStatus = "RUNNING"      // Dry run đang thực thi.
	DryrunStatusSuccess     DryrunStatus = "SUCCESS"      // Dry run pass hoàn toàn.
	DryrunStatusWarning     DryrunStatus = "WARNING"      // Dry run usable nhưng có cảnh báo cần hiển thị.
	DryrunStatusFailed      DryrunStatus = "FAILED"       // Dry run thất bại, source chưa đủ điều kiện chạy.
)

const DryrunSampleLimitDefault = 10

type CrawlMode string

const (
	CrawlModeSleep  CrawlMode = "SLEEP"  // Cào thưa khi tín hiệu thảo luận thấp.
	CrawlModeNormal CrawlMode = "NORMAL" // Tần suất mặc định cho vận hành bình thường.
	CrawlModeCrisis CrawlMode = "CRISIS" // Cào dày khi có tín hiệu khủng hoảng hoặc cần tập trung theo dõi.
)

type JobStatus string

const (
	JobStatusPending   JobStatus = "PENDING"   // Đã tạo job/task nhưng chưa bắt đầu chạy.
	JobStatusRunning   JobStatus = "RUNNING"   // Đang thực thi.
	JobStatusSuccess   JobStatus = "SUCCESS"   // Hoàn tất thành công.
	JobStatusPartial   JobStatus = "PARTIAL"   // Thành công một phần, thường dùng khi có nhiều item con.
	JobStatusFailed    JobStatus = "FAILED"    // Thất bại và cần retry/handle lỗi.
	JobStatusCancelled JobStatus = "CANCELLED" // Bị hủy do pause/archive hoặc quyết định điều phối.
)

type BatchStatus string

const (
	BatchStatusReceived   BatchStatus = "RECEIVED"   // Mới ghi nhận batch raw vào hệ thống.
	BatchStatusDownloaded BatchStatus = "DOWNLOADED" // Đã tải raw file/payload về worker.
	BatchStatusParsed     BatchStatus = "PARSED"     // Đã parse xong và sẵn sàng publish UAP.
	BatchStatusFailed     BatchStatus = "FAILED"     // Parse pipeline thất bại trước bước publish.
)

type PublishStatus string

const (
	PublishStatusPending    PublishStatus = "PENDING"    // Chưa bắt đầu publish UAP.
	PublishStatusPublishing PublishStatus = "PUBLISHING" // Đang đẩy record sang Kafka/analysis.
	PublishStatusSuccess    PublishStatus = "SUCCESS"    // Publish xong toàn batch.
	PublishStatusFailed     PublishStatus = "FAILED"     // Publish lỗi, thường cần retry full batch.
)

type TargetType string

const (
	TargetTypeKeyword TargetType = "KEYWORD"  // Crawl theo từ khóa.
	TargetTypeProfile TargetType = "PROFILE"  // Crawl theo profile/page/channel.
	TargetTypePostURL TargetType = "POST_URL" // Crawl theo link bài viết/video cụ thể.
)

type TriggerType string

const (
	TriggerTypeManual       TriggerType = "MANUAL"        // Do user hoặc internal API trigger trực tiếp.
	TriggerTypeScheduled    TriggerType = "SCHEDULED"     // Do scheduler tạo ra theo lịch.
	TriggerTypeProjectEvent TriggerType = "PROJECT_EVENT" // Do project lifecycle event phát sang.
	TriggerTypeCrisisEvent  TriggerType = "CRISIS_EVENT"  // Do adaptive crawl/crisis controller yêu cầu.
	TriggerTypeWebhookPush  TriggerType = "WEBHOOK_PUSH"  // Do dữ liệu được đẩy vào từ webhook.
)

func IsUsableDryrunStatus(status DryrunStatus) bool {
	switch status {
	case DryrunStatusSuccess, DryrunStatusWarning:
		return true
	default:
		return false
	}
}

func IsTerminalDryrunStatus(status DryrunStatus) bool {
	switch status {
	case DryrunStatusFailed, DryrunStatusWarning, DryrunStatusSuccess:
		return true
	default:
		return false
	}
}
