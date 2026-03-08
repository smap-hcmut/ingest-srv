package execution

import (
	"time"

	"ingest-srv/internal/model"
)

// DispatchTargetInput triggers one execution unit for an existing crawl target.
type DispatchTargetInput struct {
	DataSourceID string
	TargetID     string
	TriggerType  model.TriggerType
	ScheduledFor time.Time
	RequestedAt  time.Time
	CronExpr     string
}

// DispatchTargetOutput contains the created runtime lineage for a dispatch.
type DispatchTaskOutput struct {
	ExternalTaskID string `json:"external_task_id"`
	TaskID         string `json:"task_id"`
	Queue          string `json:"queue"`
	Action         string `json:"action"`
	Status         string `json:"status"`
	Keyword        string `json:"keyword,omitempty"`
	ErrorMessage   string `json:"error_message,omitempty"`
}

type DispatchTargetOutput struct {
	ScheduledJobID string `json:"scheduled_job_id"`
	Status         string `json:"status"`
	TaskCount      int    `json:"task_count"`
	PublishedCount int    `json:"published_count"`
	FailedCount    int    `json:"failed_count"`
	Tasks          []DispatchTaskOutput `json:"tasks"`
}

// DispatchTargetManuallyInput triggers one execution unit for an existing crawl target.
type DispatchTargetManuallyInput struct {
	DataSourceID string
	TargetID     string
}

// DispatchTargetManuallyOutput is the manual-dispatch compatibility output.
type DispatchTargetManuallyOutput = DispatchTargetOutput

// DispatchDueTargetsInput triggers scheduler-owned due-target dispatch.
type DispatchDueTargetsInput struct {
	Now      time.Time
	Limit    int
	CronExpr string
}

// DispatchDueTargetsOutput summarizes one scheduler tick.
type DispatchDueTargetsOutput struct {
	DueCount         int `json:"due_count"`
	ClaimedCount     int `json:"claimed_count"`
	DispatchedCount  int `json:"dispatched_count"`
	SkippedRaceCount int `json:"skipped_race_count"`
	FailedCount      int `json:"failed_count"`
}

// DispatchSpec is the normalized runtime dispatch shape built by the usecase.
type DispatchSpec struct {
	Queue  string
	Action string
	Params map[string]interface{}
	Keyword string
}

// PublishDispatchInput is the usecase-facing payload passed to the RabbitMQ producer.
type PublishDispatchInput struct {
	Queue     string
	TaskID    string
	Action    string
	Params    map[string]interface{}
	CreatedAt time.Time
}

// HandleCompletionInput is the usecase-facing completion payload consumed by ingest.
type HandleCompletionInput struct {
	TaskID        string
	Status        string
	CompletedAt   string
	StorageBucket string
	StoragePath   string
	BatchID       string
	Checksum      string
	ItemCount     *int
	Error         string
	Metadata      map[string]interface{}
}
