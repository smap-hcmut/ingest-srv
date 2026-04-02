package model

import (
	"encoding/json"
	"time"

	"ingest-srv/internal/sqlboiler"
)

type ExternalTask struct {
	ID                 string          `json:"id"`                             // Định danh nội bộ của task record trong ingest.
	SourceID           string          `json:"source_id"`                      // Source sinh ra task này.
	ProjectID          string          `json:"project_id"`                     // Denormalized để filter task theo project nhanh.
	DomainTypeCode     string          `json:"domain_type_code"`               // Snapshot domain của project tại thời điểm dispatch để downstream không cần fetch ngược.
	TargetID           string          `json:"target_id,omitempty"`            // Target tương ứng khi task được tạo theo đơn vị crawl target.
	ScheduledJobID     string          `json:"scheduled_job_id,omitempty"`     // Link về job scheduler nếu task được tạo từ lịch định kỳ.
	TaskID             string          `json:"task_id"`                        // Correlation id với RabbitMQ/crawler bên thứ 3.
	Platform           string          `json:"platform"`                       // Platform mục tiêu để publisher/consumer route đúng integration.
	TaskType           string          `json:"task_type"`                      // Phân loại tác vụ như search/comments/full_flow để xử lý response phù hợp.
	RoutingKey         string          `json:"routing_key"`                    // Routing key RabbitMQ thực tế đã publish để debug/investigate.
	RequestPayload     json.RawMessage `json:"request_payload,omitempty"`      // Snapshot request gửi đi để replay và audit.
	Status             JobStatus       `json:"status"`                         // Trạng thái runtime của task khi làm việc với crawler.
	PublishedAt        *time.Time      `json:"published_at,omitempty"`         // Mốc publish request sang RabbitMQ.
	ResponseReceivedAt *time.Time      `json:"response_received_at,omitempty"` // Mốc ingest nhận callback/response từ crawler.
	CompletedAt        *time.Time      `json:"completed_at,omitempty"`         // Mốc task kết thúc hoàn toàn.
	ErrorMessage       string          `json:"error_message,omitempty"`        // Lỗi cuối cùng nếu task fail.
	CreatedAt          time.Time       `json:"created_at"`                     // Mốc tạo record.

	Source       *DataSource   `json:"source,omitempty"`        // Relation nông để biết task thuộc source nào mà không query thêm.
	Target       *CrawlTarget  `json:"target,omitempty"`        // Relation nông tới target liên quan của task.
	ScheduledJob *ScheduledJob `json:"scheduled_job,omitempty"` // Relation nông tới job đã sinh task.
	RawBatches   []RawBatch    `json:"raw_batches,omitempty"`   // Các raw batch được tạo từ task này sau khi crawler trả kết quả.
}

func NewExternalTaskFromDB(db *sqlboiler.ExternalTask) *ExternalTask {
	return newExternalTaskFromDB(db, true)
}

func newExternalTaskFromDB(db *sqlboiler.ExternalTask, withRelations bool) *ExternalTask {
	if db == nil {
		return nil
	}

	task := &ExternalTask{
		ID:                 db.ID,
		SourceID:           db.SourceID,
		ProjectID:          db.ProjectID,
		DomainTypeCode:     db.DomainTypeCode,
		TaskID:             db.TaskID,
		Platform:           db.Platform,
		TaskType:           db.TaskType,
		RoutingKey:         db.RoutingKey,
		RequestPayload:     jsonRawFromTypes(db.RequestPayload),
		Status:             JobStatus(db.Status),
		PublishedAt:        timePtrFromNull(db.PublishedAt),
		ResponseReceivedAt: timePtrFromNull(db.ResponseReceivedAt),
		CompletedAt:        timePtrFromNull(db.CompletedAt),
		ErrorMessage:       stringFromNull(db.ErrorMessage),
		CreatedAt:          db.CreatedAt,
	}
	if db.TargetID.Valid {
		task.TargetID = db.TargetID.String
	}

	if db.ScheduledJobID.Valid {
		task.ScheduledJobID = db.ScheduledJobID.String
	}

	if withRelations && db.R != nil {
		if db.R.GetSource() != nil {
			task.Source = newDataSourceFromDB(db.R.GetSource(), false)
		}
		if db.R.GetTarget() != nil {
			task.Target = newCrawlTargetFromDB(db.R.GetTarget(), false)
		}
		if db.R.GetScheduledJob() != nil {
			task.ScheduledJob = newScheduledJobFromDB(db.R.GetScheduledJob(), false)
		}
		if batches := db.R.GetRawBatches(); len(batches) > 0 {
			task.RawBatches = make([]RawBatch, 0, len(batches))
			for _, batch := range batches {
				if mapped := newRawBatchFromDB(batch, false); mapped != nil {
					task.RawBatches = append(task.RawBatches, *mapped)
				}
			}
		}
	}

	return task
}
