package model

import (
	"encoding/json"
	"time"

	"ingest-srv/internal/sqlboiler"
)

type DataSource struct {
	ID                     string           `json:"id"`                                 // Định danh source xuyên suốt toàn bộ ingest pipeline.
	ProjectID              string           `json:"project_id"`                         // Logical FK sang project để group source theo đúng phạm vi nghiệp vụ.
	Name                   string           `json:"name"`                               // Tên hiển thị để user phân biệt nhiều source cùng loại.
	Description            string           `json:"description,omitempty"`              // Mô tả ngữ cảnh crawl/upload cho UI và audit.
	SourceType             SourceType       `json:"source_type"`                        // Xác định platform hoặc loại nguồn để chọn parser/integration phù hợp.
	SourceCategory         SourceCategory   `json:"source_category"`                    // Phân biệt source crawl với passive source để áp lifecycle đúng.
	Status                 SourceStatus     `json:"status"`                             // Trạng thái hiện tại mà API, scheduler và project sync cùng hiểu.
	Config                 json.RawMessage  `json:"config,omitempty"`                   // Cấu hình nghiệp vụ theo platform, dùng làm input chính cho runtime.
	AccountRef             json.RawMessage  `json:"account_ref,omitempty"`              // Tham chiếu page/profile/keyword/file gốc để hiển thị và trace.
	MappingRules           json.RawMessage  `json:"mapping_rules,omitempty"`            // Rule chuẩn hóa raw sang UAP cho upload/webhook/parser.
	OnboardingStatus       OnboardingStatus `json:"onboarding_status"`                  // Theo dõi tiến trình setup mapping/onboarding của passive source.
	DryrunStatus           DryrunStatus     `json:"dryrun_status"`                      // Theo dõi trạng thái validate trước khi source được activate.
	DryrunLastResultID     string           `json:"dryrun_last_result_id,omitempty"`    // Link nhanh tới kết quả dry run gần nhất cho UI/API.
	CrawlMode              *CrawlMode       `json:"crawl_mode,omitempty"`               // Mode crawl hiện hành để scheduler tính tần suất chạy.
	CrawlIntervalMinutes   *int             `json:"crawl_interval_minutes,omitempty"`   // Interval đang áp dụng thực tế cho source crawl.
	NextCrawlAt            *time.Time       `json:"next_crawl_at,omitempty"`            // Mốc đến hạn để scheduler pick source.
	LastCrawlAt            *time.Time       `json:"last_crawl_at,omitempty"`            // Thời điểm bắt đầu lần crawl gần nhất.
	LastSuccessAt          *time.Time       `json:"last_success_at,omitempty"`          // Mốc thành công gần nhất để phát hiện source bị stale.
	LastErrorAt            *time.Time       `json:"last_error_at,omitempty"`            // Mốc lỗi gần nhất để phục vụ retry/alerting.
	LastErrorMessage       string           `json:"last_error_message,omitempty"`       // Lỗi runtime gần nhất để debug nhanh ở API/UI.
	WebhookID              string           `json:"webhook_id,omitempty"`               // Public identifier để route request webhook vào đúng source.
	WebhookSecretEncrypted string           `json:"webhook_secret_encrypted,omitempty"` // Secret đã mã hóa để verify chữ ký webhook an toàn.
	CreatedBy              string           `json:"created_by,omitempty"`               // Actor tạo source để audit và phân quyền thao tác.
	ActivatedAt            *time.Time       `json:"activated_at,omitempty"`             // Mốc source bắt đầu chạy thật.
	PausedAt               *time.Time       `json:"paused_at,omitempty"`                // Mốc source bị dừng để theo dõi downtime.
	ArchivedAt             *time.Time       `json:"archived_at,omitempty"`              // Mốc archive để giữ lịch sử nhưng ngừng vận hành.
	CreatedAt              time.Time        `json:"created_at"`                         // Mốc tạo record.
	UpdatedAt              time.Time        `json:"updated_at"`                         // Mốc cập nhật gần nhất để audit/sync UI.
	DeletedAt              *time.Time       `json:"deleted_at,omitempty"`               // Soft delete marker, tránh xóa cứng khỏi lịch sử.

	DryrunLastResult *DryrunResult `json:"dryrun_last_result,omitempty"` // Relation nông tới bản dry run gần nhất để giảm thêm query.
	CrawlTargets     []CrawlTarget `json:"crawl_targets,omitempty"`      // Danh sách target-group crawl thuộc source để scheduler/dryrun làm việc per-group.
}

func NewDataSourceFromDB(db *sqlboiler.DataSource) *DataSource {
	return newDataSourceFromDB(db, true)
}

func newDataSourceFromDB(db *sqlboiler.DataSource, withRelations bool) *DataSource {
	if db == nil {
		return nil
	}

	ds := &DataSource{
		ID:               db.ID,
		ProjectID:        db.ProjectID,
		Name:             db.Name,
		Description:      stringFromNull(db.Description),
		SourceType:       SourceType(db.SourceType),
		SourceCategory:   SourceCategory(db.SourceCategory),
		Status:           SourceStatus(db.Status),
		Config:           jsonRawFromTypes(db.Config),
		AccountRef:       jsonRawFromNull(db.AccountRef),
		MappingRules:     jsonRawFromNull(db.MappingRules),
		OnboardingStatus: OnboardingStatus(db.OnboardingStatus),
		DryrunStatus:     DryrunStatus(db.DryrunStatus),
		CreatedAt:        db.CreatedAt,
		UpdatedAt:        db.UpdatedAt,
	}

	if db.DryrunLastResultID.Valid {
		ds.DryrunLastResultID = db.DryrunLastResultID.String
	}
	if db.CrawlMode.Valid {
		mode := CrawlMode(db.CrawlMode.Val)
		ds.CrawlMode = &mode
	}
	ds.CrawlIntervalMinutes = intPtrFromNull(db.CrawlIntervalMinutes)
	ds.NextCrawlAt = timePtrFromNull(db.NextCrawlAt)
	ds.LastCrawlAt = timePtrFromNull(db.LastCrawlAt)
	ds.LastSuccessAt = timePtrFromNull(db.LastSuccessAt)
	ds.LastErrorAt = timePtrFromNull(db.LastErrorAt)
	ds.LastErrorMessage = stringFromNull(db.LastErrorMessage)
	ds.WebhookID = stringFromNull(db.WebhookID)
	ds.WebhookSecretEncrypted = stringFromNull(db.WebhookSecretEncrypted)
	ds.CreatedBy = stringFromNull(db.CreatedBy)
	ds.ActivatedAt = timePtrFromNull(db.ActivatedAt)
	ds.PausedAt = timePtrFromNull(db.PausedAt)
	ds.ArchivedAt = timePtrFromNull(db.ArchivedAt)
	ds.DeletedAt = timePtrFromNull(db.DeletedAt)

	if withRelations && db.R != nil && db.R.GetDryrunLastResult() != nil {
		ds.DryrunLastResult = newDryrunResultFromDB(db.R.GetDryrunLastResult(), false)
	}
	if withRelations && db.R != nil {
		if targets := db.R.GetCrawlTargets(); len(targets) > 0 {
			ds.CrawlTargets = make([]CrawlTarget, 0, len(targets))
			for _, target := range targets {
				if mapped := newCrawlTargetFromDB(target, false); mapped != nil {
					ds.CrawlTargets = append(ds.CrawlTargets, *mapped)
				}
			}
		}
	}

	return ds
}
