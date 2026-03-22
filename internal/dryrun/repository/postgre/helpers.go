package postgre

import (
	"context"
	"database/sql"
	"time"

	dryrunRepo "ingest-srv/internal/dryrun/repository"
	"ingest-srv/internal/model"
	"ingest-srv/internal/sqlboiler"

	"github.com/aarondl/null/v8"
	"github.com/aarondl/sqlboiler/v4/boil"
)

func rollbackTx(tx *sql.Tx) {
	if tx != nil {
		_ = tx.Rollback()
	}
}

func (r *implRepository) applyResultUpdate(ctx context.Context, exec boil.ContextExecutor, row *sqlboiler.DryrunResult, opt dryrunRepo.CompleteResultOptions) error {
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

	if _, err := row.Update(ctx, exec, boil.Infer()); err != nil {
		r.l.Errorf(ctx, "dryrun.repository.applyResultUpdate.Update: %v", err)
		return dryrunRepo.ErrFailedToUpdate
	}

	return nil
}
