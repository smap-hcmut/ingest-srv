package repository

import (
	"context"

	"ingest-srv/internal/model"

	"github.com/smap-hcmut/shared-libs/go/paginator"
)

// Repository defines the data access interface for DataSource.
// Convention: One interface per domain module.
// Split by entity if module grows (DataSourceRepository + DryrunRepository).
type Repository interface {
	CreateDataSource(ctx context.Context, opt CreateDataSourceOptions) (model.DataSource, error)
	DetailDataSource(ctx context.Context, id string) (model.DataSource, error)
	GetOneDataSource(ctx context.Context, opt GetOneDataSourceOptions) (model.DataSource, error)
	GetDataSources(ctx context.Context, opt GetDataSourcesOptions) ([]model.DataSource, paginator.Paginator, error)
	ListDataSources(ctx context.Context, opt ListDataSourcesOptions) ([]model.DataSource, error)
	UpdateDataSource(ctx context.Context, opt UpdateDataSourceOptions) (model.DataSource, error)
	ArchiveDataSource(ctx context.Context, id string) error
	CountActiveTargets(ctx context.Context, dataSourceID string) (int64, error)
	CreateCrawlModeChange(ctx context.Context, opt CreateCrawlModeChangeOptions) (model.CrawlModeChange, error)

	// CrawlTarget sub-resource operations.
	CreateTarget(ctx context.Context, opt CreateTargetOptions) (model.CrawlTarget, error)
	GetTarget(ctx context.Context, opt GetTargetOptions) (model.CrawlTarget, error)
	ListTargets(ctx context.Context, opt ListTargetsOptions) ([]model.CrawlTarget, error)
	UpdateTarget(ctx context.Context, opt UpdateTargetOptions) (model.CrawlTarget, error)
	DeleteTarget(ctx context.Context, opt DeleteTargetOptions) error
}
