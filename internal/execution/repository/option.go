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

type DispatchRecord struct {
	ScheduledJob model.ScheduledJob
	ExternalTask model.ExternalTask
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

type CreateDispatchOptions struct {
	Source         model.DataSource
	Target         model.CrawlTarget
	TaskID         string
	Queue          string
	Action         string
	TriggerType    model.TriggerType
	ScheduledFor   time.Time
	CronExpr       string
	RequestPayload json.RawMessage
	JobPayload     json.RawMessage
	CreatedAt      time.Time
}

type MarkDispatchPublishedOptions struct {
	ExternalTaskID string
	ScheduledJobID string
	PublishedAt    time.Time
}

type MarkDispatchFailedOptions struct {
	SourceID       string
	TargetID       string
	ExternalTaskID string
	ScheduledJobID string
	ErrorMessage   string
	FailedAt       time.Time
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
