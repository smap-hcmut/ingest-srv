package postgre

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"time"

	dryrunRepo "ingest-srv/internal/dryrun/repository"
	"ingest-srv/internal/model"
	"ingest-srv/internal/sqlboiler"

	"github.com/smap-hcmut/shared-libs/go/paginator"

	"github.com/aarondl/null/v8"
	"github.com/aarondl/sqlboiler/v4/boil"
	"github.com/aarondl/sqlboiler/v4/queries/qm"
	"github.com/google/uuid"
)

// CreateResult inserts a dryrun result in RUNNING state.
func (r *implRepository) CreateResult(ctx context.Context, opt dryrunRepo.CreateResultOptions) (model.DryrunResult, error) {
	row := &sqlboiler.DryrunResult{
		ID:          uuid.NewString(),
		SourceID:    opt.SourceID,
		ProjectID:   opt.ProjectID,
		Status:      sqlboiler.DryrunStatus(opt.Status),
		SampleCount: opt.SampleCount,
		StartedAt:   null.TimeFrom(time.Now()),
	}

	if opt.TargetID != "" {
		row.TargetID = null.StringFrom(opt.TargetID)
	}
	if opt.JobID != "" {
		row.JobID = null.StringFrom(opt.JobID)
	}
	if opt.RequestedBy != "" {
		row.RequestedBy = null.StringFrom(opt.RequestedBy)
	}

	if err := row.Insert(ctx, r.db, boil.Infer()); err != nil {
		r.l.Errorf(ctx, "dryrun.repository.CreateResult.Insert: %v", err)
		return model.DryrunResult{}, dryrunRepo.ErrFailedToInsert
	}

	return *model.NewDryrunResultFromDB(row), nil
}

// GetByJobID returns the dryrun result correlated to one runtime task.
func (r *implRepository) GetByJobID(ctx context.Context, jobID string) (model.DryrunResult, error) {
	row, err := sqlboiler.DryrunResults(
		sqlboiler.DryrunResultWhere.JobID.EQ(null.StringFrom(jobID)),
	).One(ctx, r.db)
	if err != nil {
		if err == sql.ErrNoRows {
			return model.DryrunResult{}, dryrunRepo.ErrNotFound
		}
		r.l.Errorf(ctx, "dryrun.repository.GetByJobID.One: %v", err)
		return model.DryrunResult{}, dryrunRepo.ErrFailedToGet
	}

	return *model.NewDryrunResultFromDB(row), nil
}

// UpdateResult finalizes a dryrun result with status and payload.
func (r *implRepository) UpdateResult(ctx context.Context, opt dryrunRepo.UpdateResultOptions) (model.DryrunResult, error) {
	row, err := sqlboiler.FindDryrunResult(ctx, r.db, opt.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			return model.DryrunResult{}, dryrunRepo.ErrNotFound
		}
		r.l.Errorf(ctx, "dryrun.repository.UpdateResult.Find: %v", err)
		return model.DryrunResult{}, dryrunRepo.ErrFailedToUpdate
	}

	row.Status = sqlboiler.DryrunStatus(opt.Status)
	row.SampleCount = opt.SampleCount
	if opt.CompletedAt != nil {
		row.CompletedAt = null.TimeFrom(*opt.CompletedAt)
	} else if opt.Status == string(model.DryrunStatusFailed) ||
		opt.Status == string(model.DryrunStatusWarning) ||
		opt.Status == string(model.DryrunStatusSuccess) {
		row.CompletedAt = null.TimeFrom(time.Now())
	}
	if opt.TotalFound != nil {
		row.TotalFound = null.IntFrom(*opt.TotalFound)
	}
	if len(opt.SampleData) > 0 {
		row.SampleData = null.JSONFrom(opt.SampleData)
	}
	if len(opt.Warnings) > 0 {
		row.Warnings = null.JSONFrom(opt.Warnings)
	}
	if opt.ErrorMessage != "" {
		row.ErrorMessage = null.StringFrom(opt.ErrorMessage)
	}

	if _, err := row.Update(ctx, r.db, boil.Infer()); err != nil {
		r.l.Errorf(ctx, "dryrun.repository.UpdateResult.Update: %v", err)
		return model.DryrunResult{}, dryrunRepo.ErrFailedToUpdate
	}

	return *model.NewDryrunResultFromDB(row), nil
}

// CompleteResult finalizes one dryrun result and synchronizes datasource/target state in one transaction.
func (r *implRepository) CompleteResult(ctx context.Context, opt dryrunRepo.CompleteResultOptions) (model.DryrunResult, model.DataSource, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		r.l.Errorf(ctx, "dryrun.repository.CompleteResult.BeginTx: %v", err)
		return model.DryrunResult{}, model.DataSource{}, dryrunRepo.ErrFailedToUpdate
	}
	defer rollbackTx(tx)

	resultRow, err := sqlboiler.FindDryrunResult(ctx, tx, opt.ID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.DryrunResult{}, model.DataSource{}, dryrunRepo.ErrNotFound
		}
		r.l.Errorf(ctx, "dryrun.repository.CompleteResult.FindResult: %v", err)
		return model.DryrunResult{}, model.DataSource{}, dryrunRepo.ErrFailedToUpdate
	}

	if err := r.applyResultUpdate(ctx, tx, resultRow, opt); err != nil {
		return model.DryrunResult{}, model.DataSource{}, err
	}

	sourceRow, err := sqlboiler.FindDataSource(ctx, tx, resultRow.SourceID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.DryrunResult{}, model.DataSource{}, dryrunRepo.ErrNotFound
		}
		r.l.Errorf(ctx, "dryrun.repository.CompleteResult.FindSource: %v", err)
		return model.DryrunResult{}, model.DataSource{}, dryrunRepo.ErrFailedToUpdate
	}
	if sourceRow.DeletedAt.Valid {
		return model.DryrunResult{}, model.DataSource{}, dryrunRepo.ErrNotFound
	}

	sourceRow.DryrunStatus = sqlboiler.DryrunStatus(opt.Status)
	sourceRow.DryrunLastResultID = null.StringFrom(resultRow.ID)
	switch sourceRow.Status {
	case sqlboiler.SourceStatusPENDING, sqlboiler.SourceStatusREADY:
		if opt.Status == string(model.DryrunStatusFailed) {
			sourceRow.Status = sqlboiler.SourceStatusPENDING
		} else {
			sourceRow.Status = sqlboiler.SourceStatusREADY
		}
	}
	if _, err := sourceRow.Update(ctx, tx, boil.Whitelist(
		sqlboiler.DataSourceColumns.Status,
		sqlboiler.DataSourceColumns.DryrunStatus,
		sqlboiler.DataSourceColumns.DryrunLastResultID,
		sqlboiler.DataSourceColumns.UpdatedAt,
	)); err != nil {
		r.l.Errorf(ctx, "dryrun.repository.CompleteResult.UpdateSource: %v", err)
		return model.DryrunResult{}, model.DataSource{}, dryrunRepo.ErrFailedToUpdate
	}

	if opt.ActivateTarget && resultRow.TargetID.Valid && resultRow.TargetID.String != "" {
		targetRow, targetErr := sqlboiler.FindCrawlTarget(ctx, tx, resultRow.TargetID.String)
		if targetErr != nil {
			if !errors.Is(targetErr, sql.ErrNoRows) {
				r.l.Errorf(ctx, "dryrun.repository.CompleteResult.FindTarget: %v", targetErr)
				return model.DryrunResult{}, model.DataSource{}, dryrunRepo.ErrFailedToUpdate
			}
		} else if !targetRow.IsActive {
			targetRow.IsActive = true
			if _, targetErr = targetRow.Update(ctx, tx, boil.Whitelist(
				sqlboiler.CrawlTargetColumns.IsActive,
				sqlboiler.CrawlTargetColumns.UpdatedAt,
			)); targetErr != nil {
				r.l.Errorf(ctx, "dryrun.repository.CompleteResult.UpdateTarget: %v", targetErr)
				return model.DryrunResult{}, model.DataSource{}, dryrunRepo.ErrFailedToUpdate
			}
		}
	}

	if err := tx.Commit(); err != nil {
		r.l.Errorf(ctx, "dryrun.repository.CompleteResult.Commit: %v", err)
		return model.DryrunResult{}, model.DataSource{}, dryrunRepo.ErrFailedToUpdate
	}

	return *model.NewDryrunResultFromDB(resultRow), *model.NewDataSourceFromDB(sourceRow), nil
}

// GetLatest returns the latest dryrun result for the given filter.
func (r *implRepository) GetLatest(ctx context.Context, opt dryrunRepo.GetLatestOptions) (model.DryrunResult, error) {
	mods := []qm.QueryMod{
		sqlboiler.DryrunResultWhere.SourceID.EQ(opt.SourceID),
		qm.OrderBy(sqlboiler.DryrunResultColumns.CreatedAt + " DESC"),
	}
	if opt.TargetID != "" {
		mods = append(mods, sqlboiler.DryrunResultWhere.TargetID.EQ(null.StringFrom(opt.TargetID)))
	}

	row, err := sqlboiler.DryrunResults(mods...).One(ctx, r.db)
	if err != nil {
		if err == sql.ErrNoRows {
			return model.DryrunResult{}, dryrunRepo.ErrNotFound
		}
		r.l.Errorf(ctx, "dryrun.repository.GetLatest.One: %v", err)
		return model.DryrunResult{}, dryrunRepo.ErrFailedToGet
	}

	return *model.NewDryrunResultFromDB(row), nil
}

// ListHistory returns paginated dryrun history ordered by newest first.
func (r *implRepository) ListHistory(ctx context.Context, opt dryrunRepo.ListHistoryOptions) ([]model.DryrunResult, paginator.Paginator, error) {
	filterMods := []qm.QueryMod{
		sqlboiler.DryrunResultWhere.SourceID.EQ(opt.SourceID),
	}
	if opt.TargetID != "" {
		filterMods = append(filterMods, sqlboiler.DryrunResultWhere.TargetID.EQ(null.StringFrom(opt.TargetID)))
	}

	pq := opt.Paginator
	var (
		total    int64
		rows     sqlboiler.DryrunResultSlice
		countErr error
		queryErr error
		wg       sync.WaitGroup
	)

	wg.Go(func() {
		total, countErr = sqlboiler.DryrunResults(filterMods...).Count(ctx, r.db)
	})
	wg.Go(func() {
		mods := append(filterMods,
			qm.OrderBy(sqlboiler.DryrunResultColumns.CreatedAt+" DESC"),
			qm.Limit(int(pq.Limit)),
			qm.Offset(int(pq.Offset())),
		)
		rows, queryErr = sqlboiler.DryrunResults(mods...).All(ctx, r.db)
	})
	wg.Wait()

	if countErr != nil {
		r.l.Errorf(ctx, "dryrun.repository.ListHistory.Count: %v", countErr)
		return nil, paginator.Paginator{}, dryrunRepo.ErrFailedToList
	}
	if queryErr != nil {
		r.l.Errorf(ctx, "dryrun.repository.ListHistory.All: %v", queryErr)
		return nil, paginator.Paginator{}, dryrunRepo.ErrFailedToList
	}

	results := make([]model.DryrunResult, 0, len(rows))
	for _, row := range rows {
		results = append(results, *model.NewDryrunResultFromDB(row))
	}

	return results, paginator.Paginator{
		Total:       total,
		Count:       int64(len(results)),
		PerPage:     pq.Limit,
		CurrentPage: pq.Page,
	}, nil
}
