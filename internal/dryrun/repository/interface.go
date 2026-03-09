package repository

import (
	"context"

	"ingest-srv/internal/model"
	"ingest-srv/pkg/paginator"
)

// Repository provides persistence for dryrun results.
type Repository interface {
	CreateResult(ctx context.Context, opt CreateResultOptions) (model.DryrunResult, error)
	UpdateResult(ctx context.Context, opt UpdateResultOptions) (model.DryrunResult, error)
	GetLatest(ctx context.Context, opt GetLatestOptions) (model.DryrunResult, error)
	ListHistory(ctx context.Context, opt ListHistoryOptions) ([]model.DryrunResult, paginator.Paginator, error)
}
