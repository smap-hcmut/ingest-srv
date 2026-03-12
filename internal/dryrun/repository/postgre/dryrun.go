package postgre

import (
	"context"
	"database/sql"
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
	if opt.RequestedBy != "" {
		row.RequestedBy = null.StringFrom(opt.RequestedBy)
	}

	if err := row.Insert(ctx, r.db, boil.Infer()); err != nil {
		r.l.Errorf(ctx, "dryrun.repository.CreateResult.Insert: %v", err)
		return model.DryrunResult{}, dryrunRepo.ErrFailedToInsert
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
	row.CompletedAt = null.TimeFrom(time.Now())
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
