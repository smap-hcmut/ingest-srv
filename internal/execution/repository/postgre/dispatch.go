package postgre

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

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

func (r *implRepository) CreateScheduledJob(ctx context.Context, opt repository.CreateScheduledJobOptions) (model.ScheduledJob, error) {
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

	if strings.TrimSpace(opt.Target.ID) == "" {
		if err := jobRow.Insert(ctx, r.db, boil.Infer()); err != nil {
			r.l.Errorf(ctx, "execution.repository.CreateScheduledJob.Insert: %v", err)
			return model.ScheduledJob{}, repository.ErrCreateDispatch
		}
		return *model.NewScheduledJobFromDB(jobRow), nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		r.l.Errorf(ctx, "execution.repository.CreateScheduledJob.BeginTx: %v", err)
		return model.ScheduledJob{}, repository.ErrCreateDispatch
	}
	defer rollbackTx(tx)

	lockQuery := fmt.Sprintf(
		`SELECT id FROM "schema_ingest"."%s" WHERE id = $1 AND data_source_id = $2 FOR UPDATE`,
		sqlboiler.TableNames.CrawlTargets,
	)
	var lockedTargetID string
	if err := tx.QueryRowContext(ctx, lockQuery, opt.Target.ID, opt.Source.ID).Scan(&lockedTargetID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			r.l.Warnf(ctx, "execution.repository.CreateScheduledJob.LockTarget: not found source_id=%s target_id=%s", opt.Source.ID, opt.Target.ID)
			return model.ScheduledJob{}, repository.ErrTargetNotFound
		}
		r.l.Errorf(ctx, "execution.repository.CreateScheduledJob.LockTarget: %v", err)
		return model.ScheduledJob{}, repository.ErrCreateDispatch
	}

	runningJobExists, err := sqlboiler.ScheduledJobs(
		sqlboiler.ScheduledJobWhere.TargetID.EQ(null.StringFrom(opt.Target.ID)),
		sqlboiler.ScheduledJobWhere.Status.EQ(sqlboiler.JobStatus(model.JobStatusRunning)),
	).Exists(ctx, tx)
	if err != nil {
		r.l.Errorf(ctx, "execution.repository.CreateScheduledJob.CheckRunningScheduledJob: %v", err)
		return model.ScheduledJob{}, repository.ErrCreateDispatch
	}
	if runningJobExists {
		r.l.Warnf(ctx, "execution.repository.CreateScheduledJob: source_id=%s target_id=%s already has running job", opt.Source.ID, opt.Target.ID)
		return model.ScheduledJob{}, repository.ErrDispatchConflict
	}

	if err := jobRow.Insert(ctx, tx, boil.Infer()); err != nil {
		r.l.Errorf(ctx, "execution.repository.CreateScheduledJob.InsertTx: %v", err)
		return model.ScheduledJob{}, repository.ErrCreateDispatch
	}
	if err := tx.Commit(); err != nil {
		r.l.Errorf(ctx, "execution.repository.CreateScheduledJob.Commit: %v", err)
		return model.ScheduledJob{}, repository.ErrCreateDispatch
	}

	return *model.NewScheduledJobFromDB(jobRow), nil
}

func (r *implRepository) CreateExternalTask(ctx context.Context, opt repository.CreateExternalTaskOptions) (model.ExternalTask, error) {
	taskRow := &sqlboiler.ExternalTask{
		ID:             uuid.NewString(),
		SourceID:       opt.Source.ID,
		ProjectID:      opt.Source.ProjectID,
		DomainTypeCode: opt.DomainTypeCode,
		TaskID:         opt.TaskID,
		Platform:       strings.ToLower(string(opt.Source.SourceType)),
		TaskType:       opt.Action,
		RoutingKey:     opt.Queue,
		RequestPayload: types.JSON(opt.RequestPayload),
		Status:         sqlboiler.JobStatus(model.JobStatusPending),
	}
	if strings.TrimSpace(opt.ScheduledJobID) != "" {
		taskRow.ScheduledJobID = null.StringFrom(strings.TrimSpace(opt.ScheduledJobID))
	}
	if opt.Target.ID != "" {
		taskRow.TargetID = null.StringFrom(opt.Target.ID)
	}

	if err := taskRow.Insert(ctx, r.db, boil.Infer()); err != nil {
		r.l.Errorf(ctx, "execution.repository.CreateExternalTask.Insert: %v", err)
		return model.ExternalTask{}, repository.ErrCreateDispatch
	}

	return *model.NewExternalTaskFromDB(taskRow), nil
}

func (r *implRepository) MarkExternalTaskPublished(ctx context.Context, opt repository.MarkExternalTaskPublishedOptions) error {
	taskRow, err := sqlboiler.FindExternalTask(ctx, r.db, opt.ExternalTaskID)
	if err != nil {
		r.l.Errorf(ctx, "execution.repository.MarkExternalTaskPublished.FindTask: %v", err)
		return repository.ErrUpdateDispatch
	}

	taskRow.Status = sqlboiler.JobStatus(model.JobStatusRunning)
	taskRow.PublishedAt = null.TimeFrom(opt.PublishedAt)
	taskRow.ErrorMessage = null.String{}
	if _, err := taskRow.Update(ctx, r.db, boil.Whitelist(
		sqlboiler.ExternalTaskColumns.Status,
		sqlboiler.ExternalTaskColumns.PublishedAt,
		sqlboiler.ExternalTaskColumns.ErrorMessage,
	)); err != nil {
		r.l.Errorf(ctx, "execution.repository.MarkExternalTaskPublished.UpdateTask: %v", err)
		return repository.ErrUpdateDispatch
	}

	return nil
}

func (r *implRepository) MarkExternalTaskFailed(ctx context.Context, opt repository.MarkExternalTaskFailedOptions) error {
	taskRow, err := sqlboiler.FindExternalTask(ctx, r.db, opt.ExternalTaskID)
	if err != nil {
		r.l.Errorf(ctx, "execution.repository.MarkExternalTaskFailed.FindTask: %v", err)
		return repository.ErrUpdateDispatch
	}

	taskRow.Status = sqlboiler.JobStatus(model.JobStatusFailed)
	taskRow.CompletedAt = null.TimeFrom(opt.FailedAt)
	taskRow.ErrorMessage = null.StringFrom(opt.ErrorMessage)
	if _, err := taskRow.Update(ctx, r.db, boil.Whitelist(
		sqlboiler.ExternalTaskColumns.Status,
		sqlboiler.ExternalTaskColumns.CompletedAt,
		sqlboiler.ExternalTaskColumns.ErrorMessage,
	)); err != nil {
		r.l.Errorf(ctx, "execution.repository.MarkExternalTaskFailed.UpdateTask: %v", err)
		return repository.ErrUpdateDispatch
	}

	return nil
}

func (r *implRepository) FinalizeScheduledJob(ctx context.Context, opt repository.FinalizeScheduledJobOptions) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		r.l.Errorf(ctx, "execution.repository.FinalizeScheduledJob.BeginTx: %v", err)
		return repository.ErrUpdateDispatch
	}
	defer rollbackTx(tx)

	jobRow, err := sqlboiler.FindScheduledJob(ctx, tx, opt.ScheduledJobID)
	if err != nil {
		r.l.Errorf(ctx, "execution.repository.FinalizeScheduledJob.FindJob: %v", err)
		return repository.ErrUpdateDispatch
	}

	jobRow.Status = sqlboiler.JobStatus(opt.Status)
	if opt.CompletedAt != nil {
		jobRow.CompletedAt = null.TimeFrom(*opt.CompletedAt)
	} else {
		jobRow.CompletedAt = null.Time{}
	}
	if strings.TrimSpace(opt.ErrorMessage) == "" {
		jobRow.ErrorMessage = null.String{}
	} else {
		jobRow.ErrorMessage = null.StringFrom(opt.ErrorMessage)
	}
	if _, err := jobRow.Update(ctx, tx, boil.Whitelist(
		sqlboiler.ScheduledJobColumns.Status,
		sqlboiler.ScheduledJobColumns.CompletedAt,
		sqlboiler.ScheduledJobColumns.ErrorMessage,
	)); err != nil {
		r.l.Errorf(ctx, "execution.repository.FinalizeScheduledJob.UpdateJob: %v", err)
		return repository.ErrUpdateDispatch
	}

	if opt.Status == model.JobStatusFailed || opt.Status == model.JobStatusPartial {
		eventTime := timeNowOrCompletedAt(opt.CompletedAt)
		if err := r.updateTargetFailure(ctx, tx, opt.TargetID, strings.TrimSpace(opt.ErrorMessage), eventTime); err != nil {
			return err
		}
		if err := r.updateSourceFailure(ctx, tx, opt.SourceID, strings.TrimSpace(opt.ErrorMessage), eventTime); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		r.l.Errorf(ctx, "execution.repository.FinalizeScheduledJob.Commit: %v", err)
		return repository.ErrUpdateDispatch
	}

	return nil
}

func timeNowOrCompletedAt(value *time.Time) time.Time {
	if value != nil {
		return *value
	}
	return time.Now().UTC()
}
