package model

import (
	"encoding/json"
	"time"

	"ingest-srv/internal/sqlboiler"
)

type RawBatch struct {
	ID                 string          `json:"id"`                              // Định danh của batch raw trong ingest.
	SourceID           string          `json:"source_id"`                       // Source sở hữu batch này.
	ProjectID          string          `json:"project_id"`                      // Denormalized để query batch theo project nhanh.
	ExternalTaskID     string          `json:"external_task_id,omitempty"`      // Link về task crawler tạo ra batch; null với upload/webhook/manual.
	BatchID            string          `json:"batch_id"`                        // Dedup key giữa các lần nhận dữ liệu raw.
	Status             BatchStatus     `json:"status"`                          // Parse lifecycle hiện tại của batch.
	StorageBucket      string          `json:"storage_bucket"`                  // Bucket MinIO/S3 đang chứa raw file.
	StoragePath        string          `json:"storage_path"`                    // Đường dẫn object cụ thể để worker tải lại.
	StorageURL         string          `json:"storage_url,omitempty"`           // URL tham chiếu nếu cần debug hoặc download trực tiếp.
	ItemCount          *int            `json:"item_count,omitempty"`            // Số item thô trong batch nếu đã biết từ crawler/parser.
	SizeBytes          *int64          `json:"size_bytes,omitempty"`            // Dung lượng file để monitoring và quota.
	Checksum           string          `json:"checksum,omitempty"`              // Dùng kiểm tra integrity và hỗ trợ dedup.
	ReceivedAt         time.Time       `json:"received_at"`                     // Mốc ingest nhận batch.
	ParsedAt           *time.Time      `json:"parsed_at,omitempty"`             // Mốc parse hoàn tất.
	PublishStatus      PublishStatus   `json:"publish_status"`                  // Publish lifecycle tách riêng khỏi parse lifecycle.
	PublishRecordCount int             `json:"publish_record_count"`            // Số UAP record đã publish thành công.
	FirstEventID       string          `json:"first_event_id,omitempty"`        // Event id đầu tiên để trace sang Kafka/analysis.
	LastEventID        string          `json:"last_event_id,omitempty"`         // Event id cuối cùng để xác định range publish.
	UAPPublishedAt     *time.Time      `json:"uap_published_at,omitempty"`      // Mốc publish xong toàn batch.
	ErrorMessage       string          `json:"error_message,omitempty"`         // Lỗi parse/raw pipeline.
	PublishError       string          `json:"publish_error,omitempty"`         // Lỗi riêng của bước publish sang Kafka.
	RawMetadata        json.RawMessage `json:"raw_metadata,omitempty"`          // Metadata phụ của crawler như duration/version/flags.
	CreatedAt          time.Time       `json:"created_at"`                      // Mốc tạo record.

	Source       *DataSource   `json:"source,omitempty"`                        // Relation nông tới source sở hữu batch.
	ExternalTask *ExternalTask `json:"external_task,omitempty"`                 // Relation nông tới external task sinh ra batch.
}

func NewRawBatchFromDB(db *sqlboiler.RawBatch) *RawBatch {
	return newRawBatchFromDB(db, true)
}

func newRawBatchFromDB(db *sqlboiler.RawBatch, withRelations bool) *RawBatch {
	if db == nil {
		return nil
	}

	batch := &RawBatch{
		ID:                 db.ID,
		SourceID:           db.SourceID,
		ProjectID:          db.ProjectID,
		BatchID:            db.BatchID,
		Status:             BatchStatus(db.Status),
		StorageBucket:      db.StorageBucket,
		StoragePath:        db.StoragePath,
		StorageURL:         stringFromNull(db.StorageURL),
		ItemCount:          intPtrFromNull(db.ItemCount),
		SizeBytes:          int64PtrFromNull(db.SizeBytes),
		Checksum:           stringFromNull(db.Checksum),
		ReceivedAt:         db.ReceivedAt,
		ParsedAt:           timePtrFromNull(db.ParsedAt),
		PublishStatus:      PublishStatus(db.PublishStatus),
		PublishRecordCount: db.PublishRecordCount,
		FirstEventID:       stringFromNull(db.FirstEventID),
		LastEventID:        stringFromNull(db.LastEventID),
		UAPPublishedAt:     timePtrFromNull(db.UapPublishedAt),
		ErrorMessage:       stringFromNull(db.ErrorMessage),
		PublishError:       stringFromNull(db.PublishError),
		RawMetadata:        jsonRawFromNull(db.RawMetadata),
		CreatedAt:          db.CreatedAt,
	}

	if db.ExternalTaskID.Valid {
		batch.ExternalTaskID = db.ExternalTaskID.String
	}

	if withRelations && db.R != nil {
		if db.R.GetSource() != nil {
			batch.Source = newDataSourceFromDB(db.R.GetSource(), false)
		}
		if db.R.GetExternalTask() != nil {
			batch.ExternalTask = newExternalTaskFromDB(db.R.GetExternalTask(), false)
		}
	}

	return batch
}
