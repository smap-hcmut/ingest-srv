package postgre

import (
	"context"
	"strings"
	"time"

	"ingest-srv/internal/execution/repository"
	"ingest-srv/internal/model"
	"ingest-srv/internal/sqlboiler"

	"github.com/aarondl/null/v8"
	"github.com/aarondl/sqlboiler/v4/boil"
)

func (r *implRepository) CancelProjectRuntime(ctx context.Context, opt repository.CancelProjectRuntimeOptions) error {
	projectID := strings.TrimSpace(opt.ProjectID)
	if projectID == "" {
		return repository.ErrCancelRuntime
	}

	canceledAt := opt.CanceledAt.UTC()
	if canceledAt.IsZero() {
		canceledAt = time.Now().UTC()
	}

	reason := strings.TrimSpace(opt.Reason)
	if reason == "" {
		reason = "cancelled due to project lifecycle transition"
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		r.l.Errorf(ctx, "execution.repository.CancelProjectRuntime.BeginTx: %v", err)
		return repository.ErrCancelRuntime
	}
	defer rollbackTx(tx)

	jobRows, err := sqlboiler.ScheduledJobs(
		sqlboiler.ScheduledJobWhere.ProjectID.EQ(projectID),
		sqlboiler.ScheduledJobWhere.Status.IN([]sqlboiler.JobStatus{
			sqlboiler.JobStatus(model.JobStatusPending),
			sqlboiler.JobStatus(model.JobStatusRunning),
			sqlboiler.JobStatus(model.JobStatusPartial),
		}),
	).All(ctx, tx)
	if err != nil {
		r.l.Errorf(ctx, "execution.repository.CancelProjectRuntime.ListJobs: %v", err)
		return repository.ErrCancelRuntime
	}

	if _, err := sqlboiler.ExternalTasks(
		sqlboiler.ExternalTaskWhere.ProjectID.EQ(projectID),
		sqlboiler.ExternalTaskWhere.Status.IN([]sqlboiler.JobStatus{
			sqlboiler.JobStatus(model.JobStatusPending),
			sqlboiler.JobStatus(model.JobStatusRunning),
		}),
	).UpdateAll(ctx, tx, sqlboiler.M{
		sqlboiler.ExternalTaskColumns.Status:       sqlboiler.JobStatus(model.JobStatusCancelled),
		sqlboiler.ExternalTaskColumns.CompletedAt:  null.TimeFrom(canceledAt),
		sqlboiler.ExternalTaskColumns.ErrorMessage: null.StringFrom(reason),
	}); err != nil {
		r.l.Errorf(ctx, "execution.repository.CancelProjectRuntime.CancelTasks: %v", err)
		return repository.ErrCancelRuntime
	}

	if _, err := sqlboiler.ScheduledJobs(
		sqlboiler.ScheduledJobWhere.ProjectID.EQ(projectID),
		sqlboiler.ScheduledJobWhere.Status.IN([]sqlboiler.JobStatus{
			sqlboiler.JobStatus(model.JobStatusPending),
			sqlboiler.JobStatus(model.JobStatusRunning),
			sqlboiler.JobStatus(model.JobStatusPartial),
		}),
	).UpdateAll(ctx, tx, sqlboiler.M{
		sqlboiler.ScheduledJobColumns.Status:       sqlboiler.JobStatus(model.JobStatusCancelled),
		sqlboiler.ScheduledJobColumns.CompletedAt:  null.TimeFrom(canceledAt),
		sqlboiler.ScheduledJobColumns.ErrorMessage: null.StringFrom(reason),
	}); err != nil {
		r.l.Errorf(ctx, "execution.repository.CancelProjectRuntime.CancelJobs: %v", err)
		return repository.ErrCancelRuntime
	}

	seenTargets := make(map[string]struct{}, len(jobRows))
	for _, jobRow := range jobRows {
		if jobRow == nil || !jobRow.TargetID.Valid {
			continue
		}
		targetID := strings.TrimSpace(jobRow.TargetID.String)
		if targetID == "" {
			continue
		}
		if _, exists := seenTargets[targetID]; exists {
			continue
		}
		seenTargets[targetID] = struct{}{}

		targetRow, err := sqlboiler.FindCrawlTarget(ctx, tx, targetID)
		if err != nil {
			r.l.Errorf(ctx, "execution.repository.CancelProjectRuntime.FindTarget: target_id=%s err=%v", targetID, err)
			return repository.ErrCancelRuntime
		}

		targetRow.NextCrawlAt = null.Time{}
		targetRow.LastCrawlAt = fallbackLastCrawlAt(targetRow.LastSuccessAt, targetRow.LastErrorAt)
		if _, err := targetRow.Update(ctx, tx, boil.Whitelist(
			sqlboiler.CrawlTargetColumns.NextCrawlAt,
			sqlboiler.CrawlTargetColumns.LastCrawlAt,
			sqlboiler.CrawlTargetColumns.UpdatedAt,
		)); err != nil {
			r.l.Errorf(ctx, "execution.repository.CancelProjectRuntime.UpdateTarget: target_id=%s err=%v", targetID, err)
			return repository.ErrCancelRuntime
		}
	}

	if err := tx.Commit(); err != nil {
		r.l.Errorf(ctx, "execution.repository.CancelProjectRuntime.Commit: %v", err)
		return repository.ErrCancelRuntime
	}

	return nil
}

func fallbackLastCrawlAt(lastSuccessAt, lastErrorAt null.Time) null.Time {
	switch {
	case lastSuccessAt.Valid && (!lastErrorAt.Valid || !lastSuccessAt.Time.Before(lastErrorAt.Time)):
		return lastSuccessAt
	case lastErrorAt.Valid:
		return lastErrorAt
	default:
		return null.Time{}
	}
}
