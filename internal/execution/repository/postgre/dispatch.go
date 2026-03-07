package postgre

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"ingest-srv/internal/execution/repository"
	"ingest-srv/internal/model"
	"ingest-srv/internal/sqlboiler"

	"github.com/aarondl/null/v8"
	"github.com/aarondl/sqlboiler/v4/boil"
	"github.com/aarondl/sqlboiler/v4/queries/qm"
	"github.com/aarondl/sqlboiler/v4/types"
	"github.com/google/uuid"
)

func (r *implRepository) GetDispatchContext(ctx context.Context, dataSourceID, targetID string) (repository.DispatchContext, error) {
	sourceRow, err := sqlboiler.DataSources(
		sqlboiler.DataSourceWhere.ID.EQ(strings.TrimSpace(dataSourceID)),
	).One(ctx, r.db)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return repository.DispatchContext{}, repository.ErrDataSourceNotFound
		}
		r.l.Errorf(ctx, "execution.repository.GetDispatchContext.DataSource: %v", err)
		return repository.DispatchContext{}, repository.ErrDataSourceNotFound
	}

	targetRow, err := sqlboiler.CrawlTargets(
		sqlboiler.CrawlTargetWhere.ID.EQ(strings.TrimSpace(targetID)),
		sqlboiler.CrawlTargetWhere.DataSourceID.EQ(sourceRow.ID),
		qm.Load(sqlboiler.CrawlTargetRels.DataSource),
	).One(ctx, r.db)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return repository.DispatchContext{}, repository.ErrTargetNotFound
		}
		r.l.Errorf(ctx, "execution.repository.GetDispatchContext.Target: %v", err)
		return repository.DispatchContext{}, repository.ErrTargetNotFound
	}

	return repository.DispatchContext{
		Source: *model.NewDataSourceFromDB(sourceRow),
		Target: *model.NewCrawlTargetFromDB(targetRow),
	}, nil
}

func (r *implRepository) CreateDispatch(ctx context.Context, opt repository.CreateDispatchOptions) (repository.DispatchRecord, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		r.l.Errorf(ctx, "execution.repository.CreateDispatch.BeginTx: %v", err)
		return repository.DispatchRecord{}, repository.ErrCreateDispatch
	}
	defer rollbackTx(tx)

	triggerType := opt.TriggerType
	if triggerType == "" {
		triggerType = model.TriggerTypeManual
	}

	scheduledFor := opt.ScheduledFor
	if scheduledFor.IsZero() {
		scheduledFor = opt.CreatedAt
	}

	jobRow := &sqlboiler.ScheduledJob{
		ID:           uuid.NewString(),
		SourceID:     opt.Source.ID,
		ProjectID:    opt.Source.ProjectID,
		Status:       sqlboiler.JobStatus(model.JobStatusRunning),
		TriggerType:  sqlboiler.TriggerType(triggerType),
		CrawlMode:    sqlboiler.CrawlMode(*opt.Source.CrawlMode),
		ScheduledFor: scheduledFor,
		StartedAt:    null.TimeFrom(opt.CreatedAt),
	}
	if opt.Target.ID != "" {
		jobRow.TargetID = null.StringFrom(opt.Target.ID)
	}
	if strings.TrimSpace(opt.CronExpr) != "" {
		jobRow.CronExpr = null.StringFrom(strings.TrimSpace(opt.CronExpr))
	}
	if len(opt.JobPayload) > 0 {
		jobRow.Payload = null.JSONFrom(opt.JobPayload)
	}

	if err := jobRow.Insert(ctx, tx, boil.Infer()); err != nil {
		r.l.Errorf(ctx, "execution.repository.CreateDispatch.InsertJob: %v", err)
		return repository.DispatchRecord{}, repository.ErrCreateDispatch
	}

	taskRow := &sqlboiler.ExternalTask{
		ID:             uuid.NewString(),
		SourceID:       opt.Source.ID,
		ProjectID:      opt.Source.ProjectID,
		ScheduledJobID: null.StringFrom(jobRow.ID),
		TaskID:         opt.TaskID,
		Platform:       strings.ToLower(string(opt.Source.SourceType)),
		TaskType:       opt.Action,
		RoutingKey:     opt.Queue,
		RequestPayload: types.JSON(opt.RequestPayload),
		Status:         sqlboiler.JobStatus(model.JobStatusPending),
	}
	if opt.Target.ID != "" {
		taskRow.TargetID = null.StringFrom(opt.Target.ID)
	}

	if err := taskRow.Insert(ctx, tx, boil.Infer()); err != nil {
		r.l.Errorf(ctx, "execution.repository.CreateDispatch.InsertTask: %v", err)
		return repository.DispatchRecord{}, repository.ErrCreateDispatch
	}

	if err := tx.Commit(); err != nil {
		r.l.Errorf(ctx, "execution.repository.CreateDispatch.Commit: %v", err)
		return repository.DispatchRecord{}, repository.ErrCreateDispatch
	}

	return repository.DispatchRecord{
		ScheduledJob: *model.NewScheduledJobFromDB(jobRow),
		ExternalTask: *model.NewExternalTaskFromDB(taskRow),
	}, nil
}

func (r *implRepository) MarkDispatchPublished(ctx context.Context, opt repository.MarkDispatchPublishedOptions) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		r.l.Errorf(ctx, "execution.repository.MarkDispatchPublished.BeginTx: %v", err)
		return repository.ErrUpdateDispatch
	}
	defer rollbackTx(tx)

	taskRow, err := sqlboiler.FindExternalTask(ctx, tx, opt.ExternalTaskID)
	if err != nil {
		r.l.Errorf(ctx, "execution.repository.MarkDispatchPublished.FindTask: %v", err)
		return repository.ErrUpdateDispatch
	}
	taskRow.Status = sqlboiler.JobStatus(model.JobStatusRunning)
	taskRow.PublishedAt = null.TimeFrom(opt.PublishedAt)
	taskRow.ErrorMessage = null.String{}
	if _, err := taskRow.Update(ctx, tx, boil.Whitelist(
		sqlboiler.ExternalTaskColumns.Status,
		sqlboiler.ExternalTaskColumns.PublishedAt,
		sqlboiler.ExternalTaskColumns.ErrorMessage,
	)); err != nil {
		r.l.Errorf(ctx, "execution.repository.MarkDispatchPublished.UpdateTask: %v", err)
		return repository.ErrUpdateDispatch
	}

	jobRow, err := sqlboiler.FindScheduledJob(ctx, tx, opt.ScheduledJobID)
	if err != nil {
		r.l.Errorf(ctx, "execution.repository.MarkDispatchPublished.FindJob: %v", err)
		return repository.ErrUpdateDispatch
	}
	jobRow.Status = sqlboiler.JobStatus(model.JobStatusRunning)
	if !jobRow.StartedAt.Valid {
		jobRow.StartedAt = null.TimeFrom(opt.PublishedAt)
	}
	jobRow.ErrorMessage = null.String{}
	if _, err := jobRow.Update(ctx, tx, boil.Whitelist(
		sqlboiler.ScheduledJobColumns.Status,
		sqlboiler.ScheduledJobColumns.StartedAt,
		sqlboiler.ScheduledJobColumns.ErrorMessage,
	)); err != nil {
		r.l.Errorf(ctx, "execution.repository.MarkDispatchPublished.UpdateJob: %v", err)
		return repository.ErrUpdateDispatch
	}

	if err := tx.Commit(); err != nil {
		r.l.Errorf(ctx, "execution.repository.MarkDispatchPublished.Commit: %v", err)
		return repository.ErrUpdateDispatch
	}
	return nil
}

func (r *implRepository) MarkDispatchFailed(ctx context.Context, opt repository.MarkDispatchFailedOptions) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		r.l.Errorf(ctx, "execution.repository.MarkDispatchFailed.BeginTx: %v", err)
		return repository.ErrUpdateDispatch
	}
	defer rollbackTx(tx)

	if err := r.updateTaskFailure(ctx, tx, opt.ExternalTaskID, opt.ErrorMessage, opt.FailedAt); err != nil {
		return err
	}
	if err := r.updateJobFailure(ctx, tx, opt.ScheduledJobID, opt.ErrorMessage, opt.FailedAt); err != nil {
		return err
	}
	if err := r.updateTargetFailure(ctx, tx, opt.TargetID, opt.ErrorMessage, opt.FailedAt); err != nil {
		return err
	}
	if err := r.updateSourceFailure(ctx, tx, opt.SourceID, opt.ErrorMessage, opt.FailedAt); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		r.l.Errorf(ctx, "execution.repository.MarkDispatchFailed.Commit: %v", err)
		return repository.ErrUpdateDispatch
	}
	return nil
}
