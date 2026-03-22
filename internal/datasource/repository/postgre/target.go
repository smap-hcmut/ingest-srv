package postgre

import (
	"context"
	"database/sql"

	"ingest-srv/internal/datasource/repository"
	"ingest-srv/internal/model"
	"ingest-srv/internal/sqlboiler"

	"github.com/aarondl/null/v8"
	"github.com/aarondl/sqlboiler/v4/boil"
	"github.com/aarondl/sqlboiler/v4/queries/qm"
	"github.com/google/uuid"
)

// CreateTarget inserts a new crawl target linked to a data source.
func (r *implRepository) CreateTarget(ctx context.Context, opt repository.CreateTargetOptions) (model.CrawlTarget, error) {
	row := &sqlboiler.CrawlTarget{
		ID:           uuid.NewString(),
		DataSourceID: opt.DataSourceID,
		TargetType:   sqlboiler.TargetType(opt.TargetType),
		Values:       opt.Values,
		IsActive:     opt.IsActive,
		Priority:     opt.Priority,
	}

	if opt.Label != "" {
		row.Label = null.StringFrom(opt.Label)
	}
	if len(opt.PlatformMeta) > 0 {
		row.PlatformMeta = null.JSONFrom(opt.PlatformMeta)
	}
	if opt.CrawlIntervalMinutes > 0 {
		row.CrawlIntervalMinutes = opt.CrawlIntervalMinutes
	}

	insertColumns := []string{
		sqlboiler.CrawlTargetColumns.ID,
		sqlboiler.CrawlTargetColumns.DataSourceID,
		sqlboiler.CrawlTargetColumns.TargetType,
		sqlboiler.CrawlTargetColumns.Values,
		sqlboiler.CrawlTargetColumns.IsActive,
		sqlboiler.CrawlTargetColumns.Priority,
		sqlboiler.CrawlTargetColumns.CrawlIntervalMinutes,
	}
	if opt.Label != "" {
		insertColumns = append(insertColumns, sqlboiler.CrawlTargetColumns.Label)
	}
	if len(opt.PlatformMeta) > 0 {
		insertColumns = append(insertColumns, sqlboiler.CrawlTargetColumns.PlatformMeta)
	}

	if err := row.Insert(ctx, r.db, boil.Whitelist(insertColumns...)); err != nil {
		r.l.Errorf(ctx, "datasource.repository.CreateTarget.Insert: %v", err)
		return model.CrawlTarget{}, repository.ErrTargetFailedToInsert
	}

	return *model.NewCrawlTargetFromDB(row), nil
}

// GetTarget fetches a single crawl target by ID and datasource ownership.
func (r *implRepository) GetTarget(ctx context.Context, opt repository.GetTargetOptions) (model.CrawlTarget, error) {
	row, err := sqlboiler.CrawlTargets(
		sqlboiler.CrawlTargetWhere.ID.EQ(opt.ID),
		sqlboiler.CrawlTargetWhere.DataSourceID.EQ(opt.DataSourceID),
	).One(ctx, r.db)
	if err != nil {
		if err == sql.ErrNoRows {
			return model.CrawlTarget{}, repository.ErrTargetNotFound
		}
		r.l.Errorf(ctx, "datasource.repository.GetTarget.Find: %v", err)
		return model.CrawlTarget{}, repository.ErrTargetNotFound
	}

	return *model.NewCrawlTargetFromDB(row), nil
}

// ListTargets returns crawl targets for a data source with optional filters.
func (r *implRepository) ListTargets(ctx context.Context, opt repository.ListTargetsOptions) ([]model.CrawlTarget, error) {
	mods := []qm.QueryMod{
		sqlboiler.CrawlTargetWhere.DataSourceID.EQ(opt.DataSourceID),
		qm.OrderBy(sqlboiler.CrawlTargetColumns.Priority + " DESC, " + sqlboiler.CrawlTargetColumns.CreatedAt + " ASC"),
	}

	if opt.TargetType != "" {
		mods = append(mods, sqlboiler.CrawlTargetWhere.TargetType.EQ(sqlboiler.TargetType(opt.TargetType)))
	}
	if opt.IsActive != nil {
		mods = append(mods, sqlboiler.CrawlTargetWhere.IsActive.EQ(*opt.IsActive))
	}

	rows, err := sqlboiler.CrawlTargets(mods...).All(ctx, r.db)
	if err != nil {
		r.l.Errorf(ctx, "datasource.repository.ListTargets.All: %v", err)
		return nil, repository.ErrTargetFailedToList
	}

	targets := make([]model.CrawlTarget, 0, len(rows))
	for _, row := range rows {
		targets = append(targets, *model.NewCrawlTargetFromDB(row))
	}

	return targets, nil
}

// UpdateTarget updates a crawl target by ID. Only non-zero fields are applied.
func (r *implRepository) UpdateTarget(ctx context.Context, opt repository.UpdateTargetOptions) (model.CrawlTarget, error) {
	row, err := sqlboiler.CrawlTargets(
		sqlboiler.CrawlTargetWhere.ID.EQ(opt.ID),
		sqlboiler.CrawlTargetWhere.DataSourceID.EQ(opt.DataSourceID),
	).One(ctx, r.db)
	if err != nil {
		if err == sql.ErrNoRows {
			return model.CrawlTarget{}, repository.ErrTargetNotFound
		}
		r.l.Errorf(ctx, "datasource.repository.UpdateTarget.Find: %v", err)
		return model.CrawlTarget{}, repository.ErrTargetFailedToUpdate
	}

	if opt.Values != nil {
		row.Values = opt.Values
	}
	if opt.Label != "" {
		row.Label = null.StringFrom(opt.Label)
	}
	if len(opt.PlatformMeta) > 0 {
		row.PlatformMeta = null.JSONFrom(opt.PlatformMeta)
	}
	if opt.IsActive != nil {
		row.IsActive = *opt.IsActive
	}
	if opt.Priority != nil {
		row.Priority = *opt.Priority
	}
	if opt.CrawlIntervalMinutes != nil {
		row.CrawlIntervalMinutes = *opt.CrawlIntervalMinutes
	}

	_, err = row.Update(ctx, r.db, boil.Infer())
	if err != nil {
		r.l.Errorf(ctx, "datasource.repository.UpdateTarget.Update: %v", err)
		return model.CrawlTarget{}, repository.ErrTargetFailedToUpdate
	}

	return *model.NewCrawlTargetFromDB(row), nil
}

// DeleteTarget hard-deletes a crawl target by ID.
func (r *implRepository) DeleteTarget(ctx context.Context, opt repository.DeleteTargetOptions) error {
	row, err := sqlboiler.CrawlTargets(
		sqlboiler.CrawlTargetWhere.ID.EQ(opt.ID),
		sqlboiler.CrawlTargetWhere.DataSourceID.EQ(opt.DataSourceID),
	).One(ctx, r.db)
	if err != nil {
		if err == sql.ErrNoRows {
			return repository.ErrTargetNotFound
		}
		r.l.Errorf(ctx, "datasource.repository.DeleteTarget.Find: %v", err)
		return repository.ErrTargetFailedToDelete
	}

	_, err = row.Delete(ctx, r.db)
	if err != nil {
		r.l.Errorf(ctx, "datasource.repository.DeleteTarget.Delete: %v", err)
		return repository.ErrTargetFailedToDelete
	}

	return nil
}
