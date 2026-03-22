package postgre

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"ingest-srv/internal/execution/repository"
	"ingest-srv/internal/model"
	"ingest-srv/internal/sqlboiler"

	"github.com/aarondl/null/v8"
	"github.com/aarondl/sqlboiler/v4/boil"
	"github.com/aarondl/sqlboiler/v4/queries/qm"
	"github.com/lib/pq"
)

func (r *implRepository) updateTaskSuccess(ctx context.Context, exec boil.ContextExecutor, externalTaskID string, completedAt time.Time) error {
	if strings.TrimSpace(externalTaskID) == "" {
		return nil
	}
	row, err := sqlboiler.FindExternalTask(ctx, exec, externalTaskID)
	if err != nil {
		r.l.Errorf(ctx, "execution.repository.updateTaskSuccess.FindTask: %v", err)
		return repository.ErrCompleteTask
	}
	row.Status = sqlboiler.JobStatus(model.JobStatusSuccess)
	row.ResponseReceivedAt = null.TimeFrom(completedAt)
	row.CompletedAt = null.TimeFrom(completedAt)
	row.ErrorMessage = null.String{}
	if _, err := row.Update(ctx, exec, boil.Whitelist(
		sqlboiler.ExternalTaskColumns.Status,
		sqlboiler.ExternalTaskColumns.ResponseReceivedAt,
		sqlboiler.ExternalTaskColumns.CompletedAt,
		sqlboiler.ExternalTaskColumns.ErrorMessage,
	)); err != nil {
		r.l.Errorf(ctx, "execution.repository.updateTaskSuccess.UpdateTask: %v", err)
		return repository.ErrCompleteTask
	}
	return nil
}

func (r *implRepository) updateTaskFailure(ctx context.Context, exec boil.ContextExecutor, externalTaskID, errorMessage string, completedAt time.Time) error {
	if strings.TrimSpace(externalTaskID) == "" {
		return nil
	}
	row, err := sqlboiler.FindExternalTask(ctx, exec, externalTaskID)
	if err != nil {
		r.l.Errorf(ctx, "execution.repository.updateTaskFailure.FindTask: %v", err)
		return repository.ErrUpdateDispatch
	}
	row.Status = sqlboiler.JobStatus(model.JobStatusFailed)
	row.ResponseReceivedAt = null.TimeFrom(completedAt)
	row.CompletedAt = null.TimeFrom(completedAt)
	row.ErrorMessage = null.StringFrom(errorMessage)
	if _, err := row.Update(ctx, exec, boil.Whitelist(
		sqlboiler.ExternalTaskColumns.Status,
		sqlboiler.ExternalTaskColumns.ResponseReceivedAt,
		sqlboiler.ExternalTaskColumns.CompletedAt,
		sqlboiler.ExternalTaskColumns.ErrorMessage,
	)); err != nil {
		r.l.Errorf(ctx, "execution.repository.updateTaskFailure.UpdateTask: %v", err)
		return repository.ErrUpdateDispatch
	}
	return nil
}

func (r *implRepository) recomputeScheduledJobStatus(ctx context.Context, exec boil.ContextExecutor, scheduledJobID string, completedAt time.Time) error {
	if strings.TrimSpace(scheduledJobID) == "" {
		return nil
	}

	jobRow, err := sqlboiler.FindScheduledJob(ctx, exec, scheduledJobID)
	if err != nil {
		r.l.Errorf(ctx, "execution.repository.recomputeScheduledJobStatus.FindJob: %v", err)
		return repository.ErrCompleteTask
	}

	taskRows, err := sqlboiler.ExternalTasks(
		sqlboiler.ExternalTaskWhere.ScheduledJobID.EQ(null.StringFrom(scheduledJobID)),
		qm.OrderBy(sqlboiler.ExternalTaskColumns.CreatedAt+" ASC"),
	).All(ctx, exec)
	if err != nil {
		r.l.Errorf(ctx, "execution.repository.recomputeScheduledJobStatus.ListTasks: %v", err)
		return repository.ErrCompleteTask
	}
	if len(taskRows) == 0 {
		return nil
	}

	var successCount int
	var failedCount int
	var runningCount int
	firstError := ""
	for _, taskRow := range taskRows {
		switch model.JobStatus(taskRow.Status) {
		case model.JobStatusSuccess:
			successCount++
		case model.JobStatusFailed, model.JobStatusCancelled:
			failedCount++
			if firstError == "" && taskRow.ErrorMessage.Valid {
				firstError = taskRow.ErrorMessage.String
			}
		default:
			runningCount++
		}
	}

	expectedTaskCount := expectedDispatchTaskCount(jobRow.Payload, len(taskRows))
	missingCount := expectedTaskCount - len(taskRows)
	if missingCount > 0 {
		failedCount += missingCount
		if firstError == "" && jobRow.ErrorMessage.Valid {
			firstError = jobRow.ErrorMessage.String
		}
	}

	switch {
	case runningCount > 0 && failedCount > 0:
		jobRow.Status = sqlboiler.JobStatus(model.JobStatusPartial)
		jobRow.CompletedAt = null.Time{}
		jobRow.ErrorMessage = nullableString(firstError)
	case runningCount > 0:
		jobRow.Status = sqlboiler.JobStatus(model.JobStatusRunning)
		jobRow.CompletedAt = null.Time{}
		jobRow.ErrorMessage = null.String{}
	case successCount == expectedTaskCount:
		jobRow.Status = sqlboiler.JobStatus(model.JobStatusSuccess)
		jobRow.CompletedAt = null.TimeFrom(completedAt)
		jobRow.ErrorMessage = null.String{}
	case failedCount == expectedTaskCount:
		jobRow.Status = sqlboiler.JobStatus(model.JobStatusFailed)
		jobRow.CompletedAt = null.TimeFrom(completedAt)
		jobRow.ErrorMessage = nullableString(firstError)
	default:
		jobRow.Status = sqlboiler.JobStatus(model.JobStatusPartial)
		jobRow.CompletedAt = null.TimeFrom(completedAt)
		jobRow.ErrorMessage = nullableString(firstError)
	}

	if _, err := jobRow.Update(ctx, exec, boil.Whitelist(
		sqlboiler.ScheduledJobColumns.Status,
		sqlboiler.ScheduledJobColumns.CompletedAt,
		sqlboiler.ScheduledJobColumns.ErrorMessage,
	)); err != nil {
		r.l.Errorf(ctx, "execution.repository.recomputeScheduledJobStatus.UpdateJob: %v", err)
		return repository.ErrCompleteTask
	}

	return nil
}

func (r *implRepository) updateTargetSuccess(ctx context.Context, exec boil.ContextExecutor, targetID string, completedAt time.Time) error {
	if strings.TrimSpace(targetID) == "" {
		return nil
	}
	row, err := sqlboiler.FindCrawlTarget(ctx, exec, targetID)
	if err != nil {
		r.l.Errorf(ctx, "execution.repository.updateTargetSuccess.FindTarget: %v", err)
		return repository.ErrCompleteTask
	}
	row.LastSuccessAt = null.TimeFrom(completedAt)
	row.LastErrorAt = null.Time{}
	row.LastErrorMessage = null.String{}
	if _, err := row.Update(ctx, exec, boil.Whitelist(
		sqlboiler.CrawlTargetColumns.LastSuccessAt,
		sqlboiler.CrawlTargetColumns.LastErrorAt,
		sqlboiler.CrawlTargetColumns.LastErrorMessage,
	)); err != nil {
		r.l.Errorf(ctx, "execution.repository.updateTargetSuccess.UpdateTarget: %v", err)
		return repository.ErrCompleteTask
	}
	return nil
}

func (r *implRepository) updateTargetFailure(ctx context.Context, exec boil.ContextExecutor, targetID, errorMessage string, completedAt time.Time) error {
	if strings.TrimSpace(targetID) == "" {
		return nil
	}
	row, err := sqlboiler.FindCrawlTarget(ctx, exec, targetID)
	if err != nil {
		r.l.Errorf(ctx, "execution.repository.updateTargetFailure.FindTarget: %v", err)
		return repository.ErrUpdateDispatch
	}
	row.LastErrorAt = null.TimeFrom(completedAt)
	row.LastErrorMessage = null.StringFrom(errorMessage)
	if _, err := row.Update(ctx, exec, boil.Whitelist(
		sqlboiler.CrawlTargetColumns.LastErrorAt,
		sqlboiler.CrawlTargetColumns.LastErrorMessage,
	)); err != nil {
		r.l.Errorf(ctx, "execution.repository.updateTargetFailure.UpdateTarget: %v", err)
		return repository.ErrUpdateDispatch
	}
	return nil
}

func (r *implRepository) updateSourceSuccess(ctx context.Context, exec boil.ContextExecutor, sourceID string, completedAt time.Time) error {
	row, err := sqlboiler.FindDataSource(ctx, exec, sourceID)
	if err != nil {
		r.l.Errorf(ctx, "execution.repository.updateSourceSuccess.FindSource: %v", err)
		return repository.ErrCompleteTask
	}
	row.LastSuccessAt = null.TimeFrom(completedAt)
	row.LastErrorAt = null.Time{}
	row.LastErrorMessage = null.String{}
	if _, err := row.Update(ctx, exec, boil.Whitelist(
		sqlboiler.DataSourceColumns.LastSuccessAt,
		sqlboiler.DataSourceColumns.LastErrorAt,
		sqlboiler.DataSourceColumns.LastErrorMessage,
	)); err != nil {
		r.l.Errorf(ctx, "execution.repository.updateSourceSuccess.UpdateSource: %v", err)
		return repository.ErrCompleteTask
	}
	return nil
}

func (r *implRepository) updateSourceFailure(ctx context.Context, exec boil.ContextExecutor, sourceID, errorMessage string, completedAt time.Time) error {
	row, err := sqlboiler.FindDataSource(ctx, exec, sourceID)
	if err != nil {
		r.l.Errorf(ctx, "execution.repository.updateSourceFailure.FindSource: %v", err)
		return repository.ErrUpdateDispatch
	}
	row.LastErrorAt = null.TimeFrom(completedAt)
	row.LastErrorMessage = null.StringFrom(errorMessage)
	if _, err := row.Update(ctx, exec, boil.Whitelist(
		sqlboiler.DataSourceColumns.LastErrorAt,
		sqlboiler.DataSourceColumns.LastErrorMessage,
	)); err != nil {
		r.l.Errorf(ctx, "execution.repository.updateSourceFailure.UpdateSource: %v", err)
		return repository.ErrUpdateDispatch
	}
	return nil
}

func rollbackTx(tx *sql.Tx) {
	if tx != nil {
		_ = tx.Rollback()
	}
}

func isUniqueViolation(err error) bool {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		return string(pqErr.Code) == "23505"
	}
	return false
}

func nullableString(value string) null.String {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return null.String{}
	}
	return null.StringFrom(trimmed)
}

func expectedDispatchTaskCount(payload null.JSON, fallback int) int {
	if !payload.Valid || len(payload.JSON) == 0 {
		return fallback
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(payload.JSON, &decoded); err != nil {
		return fallback
	}
	taskCount, ok := decoded["task_count"]
	if !ok {
		return fallback
	}

	switch v := taskCount.(type) {
	case float64:
		if int(v) > 0 {
			return int(v)
		}
	case int:
		if v > 0 {
			return v
		}
	}

	return fallback
}
