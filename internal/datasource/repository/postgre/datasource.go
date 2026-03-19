package postgre

import (
	"context"
	"database/sql"
	"sync"
	"time"

	"ingest-srv/internal/datasource/repository"
	"ingest-srv/internal/model"
	"ingest-srv/internal/sqlboiler"

	"github.com/smap-hcmut/shared-libs/go/paginator"

	"github.com/aarondl/null/v8"
	"github.com/aarondl/sqlboiler/v4/boil"
	"github.com/aarondl/sqlboiler/v4/queries/qm"
	"github.com/aarondl/sqlboiler/v4/types"
	"github.com/google/uuid"
)

// CreateDataSource inserts a new data source into the database.
func (r *implRepository) CreateDataSource(ctx context.Context, opt repository.CreateDataSourceOptions) (model.DataSource, error) {
	row := &sqlboiler.DataSource{
		ID:               uuid.NewString(),
		ProjectID:        opt.ProjectID,
		Name:             opt.Name,
		SourceType:       sqlboiler.SourceType(opt.SourceType),
		SourceCategory:   sqlboiler.SourceCategory(opt.SourceCategory),
		Status:           sqlboiler.SourceStatusPENDING,
		Config:           types.JSON("{}"),
		OnboardingStatus: sqlboiler.OnboardingStatusNOT_REQUIRED,
		DryrunStatus:     sqlboiler.DryrunStatusNOT_REQUIRED,
	}

	if opt.Description != "" {
		row.Description = null.StringFrom(opt.Description)
	}
	if len(opt.Config) > 0 {
		row.Config = types.JSON(opt.Config)
	}
	if len(opt.AccountRef) > 0 {
		row.AccountRef = null.JSONFrom(opt.AccountRef)
	}
	if len(opt.MappingRules) > 0 {
		row.MappingRules = null.JSONFrom(opt.MappingRules)
	}
	if opt.CrawlMode != "" {
		row.CrawlMode = sqlboiler.NullCrawlModeFrom(sqlboiler.CrawlMode(opt.CrawlMode))
	}
	if opt.CrawlIntervalMinutes > 0 {
		row.CrawlIntervalMinutes = null.IntFrom(opt.CrawlIntervalMinutes)
	}
	if opt.WebhookID != "" {
		row.WebhookID = null.StringFrom(opt.WebhookID)
	}
	if opt.WebhookSecretEncrypted != "" {
		row.WebhookSecretEncrypted = null.StringFrom(opt.WebhookSecretEncrypted)
	}
	if opt.CreatedBy != "" {
		row.CreatedBy = null.StringFrom(opt.CreatedBy)
	}

	if err := row.Insert(ctx, r.db, boil.Infer()); err != nil {
		r.l.Errorf(ctx, "datasource.repository.CreateDataSource.Insert: %v", err)
		return model.DataSource{}, repository.ErrFailedToInsert
	}

	return *model.NewDataSourceFromDB(row), nil
}

// DetailDataSource fetches a single data source by ID (primary key lookup).
func (r *implRepository) DetailDataSource(ctx context.Context, id string) (model.DataSource, error) {
	row, err := sqlboiler.FindDataSource(ctx, r.db, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return model.DataSource{}, nil
		}
		r.l.Errorf(ctx, "datasource.repository.DetailDataSource.Find: %v", err)
		return model.DataSource{}, repository.ErrFailedToGet
	}

	// Skip soft-deleted records
	if row.DeletedAt.Valid {
		return model.DataSource{}, nil
	}

	return *model.NewDataSourceFromDB(row), nil
}

// GetOneDataSource fetches a single data source by filters (AND condition).
// Returns zero value if not found — no error.
func (r *implRepository) GetOneDataSource(ctx context.Context, opt repository.GetOneDataSourceOptions) (model.DataSource, error) {
	mods := r.buildGetOneQuery(opt)

	row, err := sqlboiler.DataSources(mods...).One(ctx, r.db)
	if err != nil {
		if err == sql.ErrNoRows {
			return model.DataSource{}, nil
		}
		r.l.Errorf(ctx, "datasource.repository.GetOneDataSource.One: %v", err)
		return model.DataSource{}, repository.ErrFailedToGet
	}

	return *model.NewDataSourceFromDB(row), nil
}

// GetDataSources fetches data sources with pagination and filters.
// Count and data queries run in parallel via goroutine.
func (r *implRepository) GetDataSources(ctx context.Context, opt repository.GetDataSourcesOptions) ([]model.DataSource, paginator.Paginator, error) {
	filterMods := r.buildGetQuery(opt)
	pq := opt.Paginator

	var (
		total    int64
		rows     sqlboiler.DataSourceSlice
		countErr error
		queryErr error
		wg       sync.WaitGroup
	)

	// Count total (parallel)
	wg.Go(func() {
		total, countErr = sqlboiler.DataSources(filterMods...).Count(ctx, r.db)
	})

	// Fetch rows (parallel)
	wg.Go(func() {
		mods := append(filterMods,
			qm.Limit(int(pq.Limit)),
			qm.Offset(int(pq.Offset())),
			qm.OrderBy(sqlboiler.DataSourceColumns.CreatedAt+" DESC"),
		)
		rows, queryErr = sqlboiler.DataSources(mods...).All(ctx, r.db)
	})

	wg.Wait()

	if countErr != nil {
		r.l.Errorf(ctx, "datasource.repository.GetDataSources.Count: %v", countErr)
		return nil, paginator.Paginator{}, repository.ErrFailedToList
	}
	if queryErr != nil {
		r.l.Errorf(ctx, "datasource.repository.GetDataSources.All: %v", queryErr)
		return nil, paginator.Paginator{}, repository.ErrFailedToList
	}

	dataSources := make([]model.DataSource, 0, len(rows))
	for _, row := range rows {
		dataSources = append(dataSources, *model.NewDataSourceFromDB(row))
	}

	pag := paginator.Paginator{
		Total:       total,
		Count:       int64(len(dataSources)),
		PerPage:     pq.Limit,
		CurrentPage: pq.Page,
	}

	return dataSources, pag, nil
}

// ListDataSources fetches data sources without pagination (for internal use).
func (r *implRepository) ListDataSources(ctx context.Context, opt repository.ListDataSourcesOptions) ([]model.DataSource, error) {
	mods := r.buildListQuery(opt)

	rows, err := sqlboiler.DataSources(mods...).All(ctx, r.db)
	if err != nil {
		r.l.Errorf(ctx, "datasource.repository.ListDataSources.All: %v", err)
		return nil, repository.ErrFailedToList
	}

	dataSources := make([]model.DataSource, 0, len(rows))
	for _, row := range rows {
		dataSources = append(dataSources, *model.NewDataSourceFromDB(row))
	}

	return dataSources, nil
}

// GetLatestDryrunByTarget returns the latest dryrun result for one crawl target.
func (r *implRepository) GetLatestDryrunByTarget(ctx context.Context, targetID string) (model.DryrunResult, error) {
	row, err := sqlboiler.DryrunResults(
		sqlboiler.DryrunResultWhere.TargetID.EQ(null.StringFrom(targetID)),
		qm.OrderBy(sqlboiler.DryrunResultColumns.CreatedAt+" DESC"),
	).One(ctx, r.db)
	if err != nil {
		if err == sql.ErrNoRows {
			return model.DryrunResult{}, nil
		}
		r.l.Errorf(ctx, "datasource.repository.GetLatestDryrunByTarget.One: target_id=%s err=%v", targetID, err)
		return model.DryrunResult{}, repository.ErrFailedToGet
	}

	return *model.NewDryrunResultFromDB(row), nil
}

// UpdateDataSource updates a data source by ID. Only non-zero fields are applied.
func (r *implRepository) UpdateDataSource(ctx context.Context, opt repository.UpdateDataSourceOptions) (model.DataSource, error) {
	row, err := sqlboiler.FindDataSource(ctx, r.db, opt.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			return model.DataSource{}, nil
		}
		r.l.Errorf(ctx, "datasource.repository.UpdateDataSource.Find: %v", err)
		return model.DataSource{}, repository.ErrFailedToGet
	}

	if opt.Name != "" {
		row.Name = opt.Name
	}
	if opt.Description != "" {
		row.Description = null.StringFrom(opt.Description)
	}
	if opt.Status != "" {
		row.Status = sqlboiler.SourceStatus(opt.Status)

		// Track lifecycle timestamps
		now := time.Now()
		switch opt.Status {
		case string(sqlboiler.SourceStatusACTIVE):
			row.ActivatedAt = null.TimeFrom(now)
		case string(sqlboiler.SourceStatusPAUSED):
			row.PausedAt = null.TimeFrom(now)
		case string(sqlboiler.SourceStatusARCHIVED):
			row.ArchivedAt = null.TimeFrom(now)
		}
	}
	if len(opt.Config) > 0 {
		row.Config = types.JSON(opt.Config)
	}
	if len(opt.AccountRef) > 0 {
		row.AccountRef = null.JSONFrom(opt.AccountRef)
	}
	if len(opt.MappingRules) > 0 {
		row.MappingRules = null.JSONFrom(opt.MappingRules)
	}
	if opt.OnboardingStatus != "" {
		row.OnboardingStatus = sqlboiler.OnboardingStatus(opt.OnboardingStatus)
	}
	if opt.DryrunStatus != "" {
		row.DryrunStatus = sqlboiler.DryrunStatus(opt.DryrunStatus)
	}
	if opt.DryrunLastResultID != "" {
		row.DryrunLastResultID = null.StringFrom(opt.DryrunLastResultID)
	}
	if opt.CrawlMode != "" {
		row.CrawlMode = sqlboiler.NullCrawlModeFrom(sqlboiler.CrawlMode(opt.CrawlMode))
	}
	if opt.CrawlIntervalMinutes != nil {
		row.CrawlIntervalMinutes = null.IntFrom(*opt.CrawlIntervalMinutes)
	}
	if opt.WebhookID != "" {
		row.WebhookID = null.StringFrom(opt.WebhookID)
	}
	if opt.WebhookSecretEncrypted != "" {
		row.WebhookSecretEncrypted = null.StringFrom(opt.WebhookSecretEncrypted)
	}
	if opt.ClearPausedAt {
		row.PausedAt = null.Time{}
	}

	_, err = row.Update(ctx, r.db, boil.Infer())
	if err != nil {
		r.l.Errorf(ctx, "datasource.repository.UpdateDataSource.Update: %v", err)
		return model.DataSource{}, repository.ErrFailedToUpdate
	}

	return *model.NewDataSourceFromDB(row), nil
}

// ArchiveDataSource soft-deletes a data source by setting deleted_at.
func (r *implRepository) ArchiveDataSource(ctx context.Context, id string) error {
	row, err := sqlboiler.FindDataSource(ctx, r.db, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return repository.ErrFailedToGet
		}
		r.l.Errorf(ctx, "datasource.repository.ArchiveDataSource.Find: %v", err)
		return repository.ErrFailedToDelete
	}

	now := time.Now()
	row.DeletedAt = null.TimeFrom(now)
	row.ArchivedAt = null.TimeFrom(now)
	row.Status = sqlboiler.SourceStatusARCHIVED

	_, err = row.Update(ctx, r.db, boil.Whitelist(
		sqlboiler.DataSourceColumns.DeletedAt,
		sqlboiler.DataSourceColumns.ArchivedAt,
		sqlboiler.DataSourceColumns.Status,
		sqlboiler.DataSourceColumns.UpdatedAt,
	))
	if err != nil {
		r.l.Errorf(ctx, "datasource.repository.ArchiveDataSource.Update: %v", err)
		return repository.ErrFailedToDelete
	}

	return nil
}

// CountActiveTargets returns the number of active crawl targets for a datasource.
func (r *implRepository) CountActiveTargets(ctx context.Context, dataSourceID string) (int64, error) {
	total, err := sqlboiler.CrawlTargets(
		sqlboiler.CrawlTargetWhere.DataSourceID.EQ(dataSourceID),
		sqlboiler.CrawlTargetWhere.IsActive.EQ(true),
	).Count(ctx, r.db)
	if err != nil {
		r.l.Errorf(ctx, "datasource.repository.CountActiveTargets.Count: %v", err)
		return 0, repository.ErrFailedToGet
	}

	return total, nil
}
