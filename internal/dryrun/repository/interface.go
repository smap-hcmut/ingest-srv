package repository

import (
	"context"

	"ingest-srv/internal/model"

	"github.com/smap-hcmut/shared-libs/go/paginator"
)

// Repository provides persistence for dryrun results.
type Repository interface {
	CreateResult(ctx context.Context, opt CreateResultOptions) (model.DryrunResult, error)
	GetByJobID(ctx context.Context, jobID string) (model.DryrunResult, error)
	UpdateResult(ctx context.Context, opt UpdateResultOptions) (model.DryrunResult, error)
	CompleteResult(ctx context.Context, opt CompleteResultOptions) (model.DryrunResult, model.DataSource, error)
	GetLatest(ctx context.Context, opt GetLatestOptions) (model.DryrunResult, error)
	ListHistory(ctx context.Context, opt ListHistoryOptions) ([]model.DryrunResult, paginator.Paginator, error)
}
