package model

import (
	"encoding/json"
	"time"

	"ingest-srv/internal/sqlboiler"

	"github.com/aarondl/sqlboiler/v4/types"
)

type CrawlTarget struct {
	ID                   string          `json:"id"`                           // Định danh target để trace dryrun/job/task theo đơn vị crawl cụ thể.
	DataSourceID         string          `json:"data_source_id"`               // Source owner của target này.
	TargetType           TargetType      `json:"target_type"`                  // Loại target (keyword/profile/post_url) quyết định cách build request.
	Values               []string        `json:"values"`                       // Nhóm giá trị crawl dùng chung một interval và cùng được thực thi như một execution unit.
	Label                string          `json:"label,omitempty"`              // Nhãn hiển thị cho UI để dễ phân biệt target.
	PlatformMeta         json.RawMessage `json:"platform_meta,omitempty"`      // Metadata mở rộng theo platform/action.
	IsActive             bool            `json:"is_active"`                    // Cho phép bật/tắt target mà không xóa lịch sử.
	Priority             int             `json:"priority"`                     // Ưu tiên xử lý khi có nhiều target đến hạn cùng lúc.
	CrawlIntervalMinutes int             `json:"crawl_interval_minutes"`       // Base interval của target trước khi apply mode multiplier.
	NextCrawlAt          *time.Time      `json:"next_crawl_at,omitempty"`      // Mốc scheduler dùng để pick target.
	LastCrawlAt          *time.Time      `json:"last_crawl_at,omitempty"`      // Mốc crawl gần nhất của target.
	LastSuccessAt        *time.Time      `json:"last_success_at,omitempty"`    // Mốc thành công gần nhất để health-check target.
	LastErrorAt          *time.Time      `json:"last_error_at,omitempty"`      // Mốc lỗi gần nhất để quyết định retry/escalation.
	LastErrorMessage     string          `json:"last_error_message,omitempty"` // Lỗi gần nhất của target.
	CreatedAt            time.Time       `json:"created_at"`                   // Mốc tạo bản ghi target.
	UpdatedAt            time.Time       `json:"updated_at"`                   // Mốc cập nhật gần nhất.

	Source *DataSource `json:"source,omitempty"` // Relation nông tới source owner.
}

func NewCrawlTargetFromDB(db *sqlboiler.CrawlTarget) *CrawlTarget {
	return newCrawlTargetFromDB(db, true)
}

func newCrawlTargetFromDB(db *sqlboiler.CrawlTarget, withRelations bool) *CrawlTarget {
	if db == nil {
		return nil
	}

	target := &CrawlTarget{
		ID:                   db.ID,
		DataSourceID:         db.DataSourceID,
		TargetType:           TargetType(db.TargetType),
		Values:               stringSliceFromTypesJSON(db.Values),
		Label:                stringFromNull(db.Label),
		PlatformMeta:         jsonRawFromNull(db.PlatformMeta),
		IsActive:             db.IsActive,
		Priority:             db.Priority,
		CrawlIntervalMinutes: db.CrawlIntervalMinutes,
		NextCrawlAt:          timePtrFromNull(db.NextCrawlAt),
		LastCrawlAt:          timePtrFromNull(db.LastCrawlAt),
		LastSuccessAt:        timePtrFromNull(db.LastSuccessAt),
		LastErrorAt:          timePtrFromNull(db.LastErrorAt),
		LastErrorMessage:     stringFromNull(db.LastErrorMessage),
		CreatedAt:            db.CreatedAt,
		UpdatedAt:            db.UpdatedAt,
	}

	if withRelations && db.R != nil && db.R.GetDataSource() != nil {
		target.Source = newDataSourceFromDB(db.R.GetDataSource(), false)
	}

	return target
}

func TypesJSONFromStringSlice(values []string) types.JSON {
	encoded, err := json.Marshal(values)
	if err != nil {
		return types.JSON([]byte("[]"))
	}
	return types.JSON(encoded)
}
