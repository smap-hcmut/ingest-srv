package repository

import (
	"encoding/json"
	"time"

	"github.com/smap-hcmut/shared-libs/go/paginator"
)

// CreateResultOptions contains data needed to insert a dryrun result.
type CreateResultOptions struct {
	SourceID    string
	ProjectID   string
	TargetID    string
	JobID       string
	Status      string
	RequestedBy string
	SampleCount int
}

// UpdateResultOptions contains data needed to finalize a dryrun result.
type UpdateResultOptions struct {
	ID           string
	Status       string
	SampleCount  int
	CompletedAt  *time.Time
	TotalFound   *int
	SampleData   json.RawMessage
	Warnings     json.RawMessage
	ErrorMessage string
}

// CompleteResultOptions atomically finalizes one dryrun result and syncs datasource/target snapshots.
type CompleteResultOptions struct {
	ID             string
	Status         string
	SampleCount    int
	CompletedAt    *time.Time
	TotalFound     *int
	SampleData     json.RawMessage
	Warnings       json.RawMessage
	ErrorMessage   string
	ActivateTarget bool
}

// GetLatestOptions contains filters for retrieving the latest dryrun result.
type GetLatestOptions struct {
	SourceID string
	TargetID string
}

// ListHistoryOptions contains filters and pagination for history queries.
type ListHistoryOptions struct {
	SourceID  string
	TargetID  string
	Paginator paginator.PaginateQuery
}
