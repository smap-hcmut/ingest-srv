package repository

import (
	"encoding/json"

	"github.com/smap-hcmut/shared-libs/go/paginator"
)

// CreateResultOptions contains data needed to insert a dryrun result.
type CreateResultOptions struct {
	SourceID    string
	ProjectID   string
	TargetID    string
	Status      string
	RequestedBy string
	SampleCount int
}

// UpdateResultOptions contains data needed to finalize a dryrun result.
type UpdateResultOptions struct {
	ID           string
	Status       string
	SampleCount  int
	TotalFound   *int
	SampleData   json.RawMessage
	Warnings     json.RawMessage
	ErrorMessage string
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
