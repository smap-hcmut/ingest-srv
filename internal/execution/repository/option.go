package repository

import (
	"encoding/json"
	"time"

	"ingest-srv/internal/model"
)

type DispatchContext struct {
	Source model.DataSource
	Target model.CrawlTarget
}

type DueTarget struct {
	Source model.DataSource
	Target model.CrawlTarget
}

type CompletionContext struct {
	ExternalTask model.ExternalTask
}

type ClaimTargetOptions struct {
	SourceID    string
	TargetID    string
	ClaimedAt   time.Time
	NextCrawlAt time.Time
}

type ReleaseClaimTargetOptions struct {
	SourceID string
	TargetID string
}

type CreateScheduledJobOptions struct {
	Source       model.DataSource
	Target       model.CrawlTarget
	TriggerType  model.TriggerType
	ScheduledFor time.Time
	CronExpr     string
	JobPayload   json.RawMessage
	CreatedAt    time.Time
}

type CreateExternalTaskOptions struct {
	Source         model.DataSource
	Target         model.CrawlTarget
	ScheduledJobID string
	TaskID         string
	Queue          string
	Action         string
	RequestPayload json.RawMessage
	CreatedAt      time.Time
}

type MarkExternalTaskPublishedOptions struct {
	ExternalTaskID string
	PublishedAt    time.Time
}

type MarkExternalTaskFailedOptions struct {
	ExternalTaskID string
	ErrorMessage   string
	FailedAt       time.Time
}

type FinalizeScheduledJobOptions struct {
	ScheduledJobID string
	SourceID       string
	TargetID       string
	Status         model.JobStatus
	ErrorMessage   string
	CompletedAt    *time.Time
}

type CancelProjectRuntimeOptions struct {
	ProjectID  string
	Reason     string
	CanceledAt time.Time
}

type CompleteTaskSuccessOptions struct {
	CompletionContext CompletionContext
	BatchID           string
	StorageBucket     string
	StoragePath       string
	Checksum          string
	ItemCount         *int
	SizeBytes         *int64
	RawMetadata       json.RawMessage
	CompletedAt       time.Time
}

type CompleteTaskErrorOptions struct {
	CompletionContext CompletionContext
	ErrorMessage      string
	CompletedAt       time.Time
}
