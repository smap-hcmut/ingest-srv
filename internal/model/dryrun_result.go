package model

import (
	"encoding/json"
	"time"

	"ingest-srv/internal/sqlboiler"
)

type DryrunResult struct {
	ID           string          `json:"id"`                        // Định danh của một lần dry run để tra cứu/audit.
	SourceID     string          `json:"source_id"`                 // Source được đem đi dry run.
	ProjectID    string          `json:"project_id"`                // Denormalized để query lịch sử dry run theo project.
	TargetID     string          `json:"target_id,omitempty"`       // Target tương ứng khi dry run per-target; null với FILE_UPLOAD/WEBHOOK.
	JobID        string          `json:"job_id,omitempty"`          // Correlation id tới execution nội bộ hoặc task bên ngoài.
	Status       DryrunStatus    `json:"status"`                    // Kết quả cuối cùng của dry run.
	SampleCount  int             `json:"sample_count"`              // Số mẫu thực tế lấy được để đánh giá source có usable hay không.
	TotalFound   *int            `json:"total_found,omitempty"`     // Tổng item tìm thấy nếu integration trả về được số này.
	SampleData   json.RawMessage `json:"sample_data,omitempty"`     // Payload mẫu cho user review trước khi activate.
	Warnings     json.RawMessage `json:"warnings,omitempty"`        // Cảnh báo không block flow nhưng cần hiển thị.
	ErrorMessage string          `json:"error_message,omitempty"`   // Lỗi chính khi dry run thất bại.
	RequestedBy  string          `json:"requested_by,omitempty"`    // Actor trigger dry run để audit.
	StartedAt    *time.Time      `json:"started_at,omitempty"`      // Mốc bắt đầu chạy để đo latency.
	CompletedAt  *time.Time      `json:"completed_at,omitempty"`    // Mốc kết thúc để xác định runtime duration.
	CreatedAt    time.Time       `json:"created_at"`                // Mốc persist result trong DB.

	Source *DataSource  `json:"source,omitempty"`                   // Relation nông để trả kèm context source khi cần.
	Target *CrawlTarget `json:"target,omitempty"`                   // Relation nông tới target đã được dry run.
}

func NewDryrunResultFromDB(db *sqlboiler.DryrunResult) *DryrunResult {
	return newDryrunResultFromDB(db, true)
}

func newDryrunResultFromDB(db *sqlboiler.DryrunResult, withRelations bool) *DryrunResult {
	if db == nil {
		return nil
	}

	dr := &DryrunResult{
		ID:           db.ID,
		SourceID:     db.SourceID,
		ProjectID:    db.ProjectID,
		JobID:        stringFromNull(db.JobID),
		Status:       DryrunStatus(db.Status),
		SampleCount:  db.SampleCount,
		TotalFound:   intPtrFromNull(db.TotalFound),
		SampleData:   jsonRawFromNull(db.SampleData),
		Warnings:     jsonRawFromNull(db.Warnings),
		ErrorMessage: stringFromNull(db.ErrorMessage),
		RequestedBy:  stringFromNull(db.RequestedBy),
		StartedAt:    timePtrFromNull(db.StartedAt),
		CompletedAt:  timePtrFromNull(db.CompletedAt),
		CreatedAt:    db.CreatedAt,
	}
	if db.TargetID.Valid {
		dr.TargetID = db.TargetID.String
	}

	if withRelations && db.R != nil {
		if db.R.GetSource() != nil {
			dr.Source = newDataSourceFromDB(db.R.GetSource(), false)
		}
		if db.R.GetTarget() != nil {
			dr.Target = newCrawlTargetFromDB(db.R.GetTarget(), false)
		}
	}

	return dr
}
