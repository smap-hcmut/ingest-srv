package dryrun

import (
	"ingest-srv/internal/model"

	"github.com/smap-hcmut/shared-libs/go/paginator"
)

// TriggerInput triggers one validation-only dryrun. For crawl sources, TargetID refers to one grouped crawl target.
type TriggerInput struct {
	SourceID    string
	TargetID    string
	SampleLimit *int
	Force       bool
}

// TriggerOutput returns the persisted dryrun result and updated datasource snapshot.
type TriggerOutput struct {
	Result     model.DryrunResult
	DataSource model.DataSource
}

// GetLatestInput retrieves the latest dryrun result for a source or source-target(group) pair.
type GetLatestInput struct {
	SourceID string
	TargetID string
}

// GetLatestOutput contains the latest dryrun result.
type GetLatestOutput struct {
	Result model.DryrunResult
}

// ListHistoryInput retrieves paginated dryrun history.
type ListHistoryInput struct {
	SourceID  string
	TargetID  string
	Paginator paginator.PaginateQuery
}

// ListHistoryOutput contains paginated dryrun history.
type ListHistoryOutput struct {
	Results   []model.DryrunResult
	Paginator paginator.Paginator
}
