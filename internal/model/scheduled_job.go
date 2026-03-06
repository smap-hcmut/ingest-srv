package model

import (
	"encoding/json"
	"time"

	"ingest-srv/internal/sqlboiler"
)

type ScheduledJob struct {
	ID           string          `json:"id"`                              // Định danh của một lần scheduler quyết định chạy source.
	SourceID     string          `json:"source_id"`                       // Source mà job này phục vụ.
	ProjectID    string          `json:"project_id"`                      // Denormalized để query job theo project không cần join.
	TargetID     string          `json:"target_id,omitempty"`             // Target tương ứng khi scheduler chạy per-target.
	Status       JobStatus       `json:"status"`                          // Trạng thái runtime hiện tại của job.
	TriggerType  TriggerType     `json:"trigger_type"`                    // Cho biết job sinh ra do lịch, event hay thao tác tay.
	CronExpr     string          `json:"cron_expr,omitempty"`             // Snapshot cron gốc để audit/debug lịch chạy.
	CrawlMode    CrawlMode       `json:"crawl_mode"`                      // Snapshot crawl mode tại thời điểm tạo job.
	ScheduledFor time.Time       `json:"scheduled_for"`                   // Thời điểm job được lên lịch phải chạy.
	StartedAt    *time.Time      `json:"started_at,omitempty"`            // Mốc worker bắt đầu xử lý job.
	CompletedAt  *time.Time      `json:"completed_at,omitempty"`          // Mốc job hoàn tất.
	RetryCount   int             `json:"retry_count"`                     // Số lần retry đã dùng để phục vụ backoff.
	ErrorMessage string          `json:"error_message,omitempty"`         // Lỗi cuối cùng nếu job fail hoặc bị cancel bất thường.
	Payload      json.RawMessage `json:"payload,omitempty"`               // Snapshot input để replay/debug job mà không cần query lại state hiện tại.
	CreatedAt    time.Time       `json:"created_at"`                      // Mốc tạo job record.

	Source        *DataSource    `json:"source,omitempty"`                // Relation nông tới source gốc.
	Target        *CrawlTarget   `json:"target,omitempty"`                // Relation nông tới target mà job đang chạy.
	ExternalTasks []ExternalTask `json:"external_tasks,omitempty"`        // Các task RabbitMQ được job này sinh ra.
}

func NewScheduledJobFromDB(db *sqlboiler.ScheduledJob) *ScheduledJob {
	return newScheduledJobFromDB(db, true)
}

func newScheduledJobFromDB(db *sqlboiler.ScheduledJob, withRelations bool) *ScheduledJob {
	if db == nil {
		return nil
	}

	job := &ScheduledJob{
		ID:           db.ID,
		SourceID:     db.SourceID,
		ProjectID:    db.ProjectID,
		Status:       JobStatus(db.Status),
		TriggerType:  TriggerType(db.TriggerType),
		CronExpr:     stringFromNull(db.CronExpr),
		CrawlMode:    CrawlMode(db.CrawlMode),
		ScheduledFor: db.ScheduledFor,
		StartedAt:    timePtrFromNull(db.StartedAt),
		CompletedAt:  timePtrFromNull(db.CompletedAt),
		RetryCount:   db.RetryCount,
		ErrorMessage: stringFromNull(db.ErrorMessage),
		Payload:      jsonRawFromNull(db.Payload),
		CreatedAt:    db.CreatedAt,
	}
	if db.TargetID.Valid {
		job.TargetID = db.TargetID.String
	}

	if withRelations && db.R != nil {
		if db.R.GetSource() != nil {
			job.Source = newDataSourceFromDB(db.R.GetSource(), false)
		}
		if db.R.GetTarget() != nil {
			job.Target = newCrawlTargetFromDB(db.R.GetTarget(), false)
		}
		if tasks := db.R.GetExternalTasks(); len(tasks) > 0 {
			job.ExternalTasks = make([]ExternalTask, 0, len(tasks))
			for _, task := range tasks {
				if mapped := newExternalTaskFromDB(task, false); mapped != nil {
					job.ExternalTasks = append(job.ExternalTasks, *mapped)
				}
			}
		}
	}

	return job
}
