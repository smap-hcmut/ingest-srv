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
	ExternalTaskID string
	TaskID         string
	Queue          string
	Action         string
	Status         string
	Keyword        string
	ErrorMessage   string
}

type DispatchTargetOutput struct {
	ScheduledJobID string
	Status         string
	TaskCount      int
	PublishedCount int
	FailedCount    int
	Tasks          []DispatchTaskOutput
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
	DueCount         int
	ClaimedCount     int
	DispatchedCount  int
	SkippedRaceCount int
	FailedCount      int
}

type CancelProjectRuntimeInput struct {
	ProjectID  string
	Reason     string
	CanceledAt time.Time
}

type QueueName string

type ActionName string

const (
	ActionNamePostDetail ActionName = "post_detail"
	ActionNameFullFlow   ActionName = "full_flow"
)

const (
	MinioVerifyRetryAttempts   = 3
	DefaultMinIntervalMinute   = 1
	DefaultMaxIntervalMinute   = 1440
	NormalModeMultiplier       = 1.0
	CrisisModeMultiplier       = 0.2
	SleepModeMultiplier        = 5.0
	TikTokFullFlowLimit        = 50
	TikTokFullFlowThreshold    = 0.3
	TikTokFullFlowCommentCount = 500
)

// DispatchSpec is the normalized runtime dispatch shape built by the usecase.
type DispatchSpec struct {
	Queue   QueueName
	Action  ActionName
	Params  map[string]interface{}
	Keyword string
}

// PublishDispatchInput is the usecase-facing payload passed to the RabbitMQ producer.
type PublishDispatchInput struct {
	Queue     QueueName
	TaskID    string
	Action    ActionName
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
