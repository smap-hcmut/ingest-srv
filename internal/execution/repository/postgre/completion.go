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
	"github.com/google/uuid"
)

func (r *implRepository) GetCompletionContext(ctx context.Context, taskID string) (repository.CompletionContext, error) {
	taskRow, err := sqlboiler.ExternalTasks(
		sqlboiler.ExternalTaskWhere.TaskID.EQ(strings.TrimSpace(taskID)),
		qm.Load(sqlboiler.ExternalTaskRels.Source),
		qm.Load(sqlboiler.ExternalTaskRels.Target),
		qm.Load(sqlboiler.ExternalTaskRels.ScheduledJob),
	).One(ctx, r.db)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return repository.CompletionContext{}, repository.ErrExternalTaskNotFound
		}
		r.l.Errorf(ctx, "execution.repository.GetCompletionContext.Task: %v", err)
		return repository.CompletionContext{}, repository.ErrGetCompletionTask
	}

	return repository.CompletionContext{
		ExternalTask: *model.NewExternalTaskFromDB(taskRow),
	}, nil
}

func (r *implRepository) HasRawBatch(ctx context.Context, sourceID, batchID string) (bool, error) {
	exists, err := sqlboiler.RawBatches(
		sqlboiler.RawBatchWhere.SourceID.EQ(strings.TrimSpace(sourceID)),
		sqlboiler.RawBatchWhere.BatchID.EQ(strings.TrimSpace(batchID)),
	).Exists(ctx, r.db)
	if err != nil {
		r.l.Errorf(ctx, "execution.repository.HasRawBatch: %v", err)
		return false, repository.ErrGetCompletionTask
	}
	return exists, nil
}

func (r *implRepository) CompleteTaskSuccess(ctx context.Context, opt repository.CompleteTaskSuccessOptions) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		r.l.Errorf(ctx, "execution.repository.CompleteTaskSuccess.BeginTx: %v", err)
		return repository.ErrCompleteTask
	}
	defer rollbackTx(tx)

	row := &sqlboiler.RawBatch{
		ID:             uuid.NewString(),
		SourceID:       opt.CompletionContext.ExternalTask.SourceID,
		ProjectID:      opt.CompletionContext.ExternalTask.ProjectID,
		BatchID:        opt.BatchID,
		Status:         sqlboiler.BatchStatus(model.BatchStatusReceived),
		StorageBucket:  opt.StorageBucket,
		StoragePath:    opt.StoragePath,
		ReceivedAt:     opt.CompletedAt,
		PublishStatus:  sqlboiler.PublishStatus(model.PublishStatusPending),
		ExternalTaskID: null.StringFrom(opt.CompletionContext.ExternalTask.ID),
	}
	if opt.ItemCount != nil {
		row.ItemCount = null.IntFrom(*opt.ItemCount)
	}
	if opt.SizeBytes != nil {
		row.SizeBytes = null.Int64From(*opt.SizeBytes)
	}
	if opt.Checksum != "" {
		row.Checksum = null.StringFrom(opt.Checksum)
	}
	if len(opt.RawMetadata) > 0 {
		row.RawMetadata = null.JSONFrom(opt.RawMetadata)
	}

	if err := row.Insert(ctx, tx, boil.Infer()); err != nil {
		if isUniqueViolation(err) {
			return repository.ErrRawBatchAlreadyExists
		}
		r.l.Errorf(ctx, "execution.repository.CompleteTaskSuccess.InsertRawBatch: %v", err)
		return repository.ErrCompleteTask
	}

	if err := r.updateTaskSuccess(ctx, tx, opt.CompletionContext.ExternalTask.ID, opt.CompletedAt); err != nil {
		return err
	}
	if err := r.updateJobSuccess(ctx, tx, opt.CompletionContext.ExternalTask.ScheduledJobID, opt.CompletedAt); err != nil {
		return err
	}
	if err := r.updateTargetSuccess(ctx, tx, opt.CompletionContext.ExternalTask.TargetID, opt.CompletedAt); err != nil {
		return err
	}
	if err := r.updateSourceSuccess(ctx, tx, opt.CompletionContext.ExternalTask.SourceID, opt.CompletedAt); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		r.l.Errorf(ctx, "execution.repository.CompleteTaskSuccess.Commit: %v", err)
		return repository.ErrCompleteTask
	}
	return nil
}

func (r *implRepository) CompleteTaskError(ctx context.Context, opt repository.CompleteTaskErrorOptions) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		r.l.Errorf(ctx, "execution.repository.CompleteTaskError.BeginTx: %v", err)
		return repository.ErrCompleteTask
	}
	defer rollbackTx(tx)

	if err := r.updateTaskFailure(ctx, tx, opt.CompletionContext.ExternalTask.ID, opt.ErrorMessage, opt.CompletedAt); err != nil {
		return err
	}
	if err := r.updateJobFailure(ctx, tx, opt.CompletionContext.ExternalTask.ScheduledJobID, opt.ErrorMessage, opt.CompletedAt); err != nil {
		return err
	}
	if err := r.updateTargetFailure(ctx, tx, opt.CompletionContext.ExternalTask.TargetID, opt.ErrorMessage, opt.CompletedAt); err != nil {
		return err
	}
	if err := r.updateSourceFailure(ctx, tx, opt.CompletionContext.ExternalTask.SourceID, opt.ErrorMessage, opt.CompletedAt); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		r.l.Errorf(ctx, "execution.repository.CompleteTaskError.Commit: %v", err)
		return repository.ErrCompleteTask
	}
	return nil
}
