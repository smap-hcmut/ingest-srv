package rabbitmq

import (
	"encoding/json"
	"time"

	"ingest-srv/internal/dryrun"
)

type DispatchMessage struct {
	TaskID    string                 `json:"task_id"`
	Action    string                 `json:"action"`
	Params    map[string]interface{} `json:"params"`
	CreatedAt string                 `json:"created_at"`
}

type CompletionMessage struct {
	TaskID        string                 `json:"task_id"`
	Queue         string                 `json:"queue"`
	Platform      string                 `json:"platform"`
	Action        string                 `json:"action"`
	Status        string                 `json:"status"`
	CompletedAt   string                 `json:"completed_at"`
	StorageBucket string                 `json:"storage_bucket"`
	StoragePath   string                 `json:"storage_path"`
	BatchID       string                 `json:"batch_id"`
	Checksum      string                 `json:"checksum"`
	ItemCount     *int                   `json:"item_count"`
	Error         string                 `json:"error"`
	Metadata      map[string]interface{} `json:"metadata"`
}

func NewDispatchMessage(input dryrun.PublishDispatchInput) DispatchMessage {
	return DispatchMessage{
		TaskID:    input.TaskID,
		Action:    string(input.Action),
		Params:    input.Params,
		CreatedAt: input.CreatedAt.Format(time.RFC3339),
	}
}

func MarshalDispatchMessage(input dryrun.PublishDispatchInput) ([]byte, error) {
	return json.Marshal(NewDispatchMessage(input))
}

func (m CompletionMessage) ToHandleCompletionInput() dryrun.HandleCompletionInput {
	return dryrun.HandleCompletionInput{
		TaskID:        m.TaskID,
		Status:        m.Status,
		CompletedAt:   m.CompletedAt,
		StorageBucket: m.StorageBucket,
		StoragePath:   m.StoragePath,
		BatchID:       m.BatchID,
		Checksum:      m.Checksum,
		ItemCount:     m.ItemCount,
		Error:         m.Error,
		Metadata:      m.Metadata,
	}
}
