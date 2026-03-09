package model

import (
	"time"

	"ingest-srv/internal/sqlboiler"
)

type CrawlModeChange struct {
	ID                  string      `json:"id"`                              // Định danh bản ghi audit của một lần đổi crawl mode.
	SourceID            string      `json:"source_id"`                       // Cho biết source nào bị đổi mode để trace đúng target.
	ProjectID           string      `json:"project_id"`                      // Denormalized để query toàn bộ lịch sử mode change theo project.
	TriggerType         TriggerType `json:"trigger_type"`                    // Phân biệt do scheduler, project event, crisis hay thao tác tay.
	FromMode            CrawlMode   `json:"from_mode"`                       // Ghi lại mode cũ để thấy transition đầy đủ.
	ToMode              CrawlMode   `json:"to_mode"`                         // Ghi lại mode mới để worker/scheduler biết target state.
	FromIntervalMinutes int         `json:"from_interval_minutes"`           // Snapshot interval cũ để audit và rollback/debug.
	ToIntervalMinutes   int         `json:"to_interval_minutes"`             // Snapshot interval mới thực tế sau khi đổi mode.
	Reason              string      `json:"reason,omitempty"`                // Mô tả ngắn lý do đổi mode để đọc log không cần mở event gốc.
	EventRef            string      `json:"event_ref,omitempty"`             // Tham chiếu event/alert/request gốc để trace ngược.
	TriggeredBy         string      `json:"triggered_by,omitempty"`          // Actor gây ra thay đổi, có thể là user hoặc service nội bộ.
	TriggeredAt         time.Time   `json:"triggered_at"`                    // Thời điểm thay đổi thực sự xảy ra trong runtime.
	CreatedAt           time.Time   `json:"created_at"`                      // Mốc tạo record để audit persistence.
	Source              *DataSource `json:"source,omitempty"`                // Relation nông để trả về context source khi cần.
}

func NewCrawlModeChangeFromDB(db *sqlboiler.CrawlModeChange) *CrawlModeChange {
	return newCrawlModeChangeFromDB(db, true)
}

func newCrawlModeChangeFromDB(db *sqlboiler.CrawlModeChange, withRelations bool) *CrawlModeChange {
	if db == nil {
		return nil
	}

	change := &CrawlModeChange{
		ID:                  db.ID,
		SourceID:            db.SourceID,
		ProjectID:           db.ProjectID,
		TriggerType:         TriggerType(db.TriggerType),
		FromMode:            CrawlMode(db.FromMode),
		ToMode:              CrawlMode(db.ToMode),
		FromIntervalMinutes: db.FromIntervalMinutes,
		ToIntervalMinutes:   db.ToIntervalMinutes,
		Reason:              stringFromNull(db.Reason),
		EventRef:            stringFromNull(db.EventRef),
		TriggeredBy:         stringFromNull(db.TriggeredBy),
		TriggeredAt:         db.TriggeredAt,
		CreatedAt:           db.CreatedAt,
	}

	if withRelations && db.R != nil && db.R.GetSource() != nil {
		change.Source = newDataSourceFromDB(db.R.GetSource(), false)
	}

	return change
}
