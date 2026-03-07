package usecase

import (
	"context"
	"encoding/json"
	"time"

	"ingest-srv/internal/execution"
	repo "ingest-srv/internal/execution/repository"
	"ingest-srv/internal/model"

	"github.com/google/uuid"
)

func (uc *implUseCase) DispatchTarget(ctx context.Context, input execution.DispatchTargetInput) (execution.DispatchTargetOutput, error) {
	dispatchCtx, err := uc.repo.GetDispatchContext(ctx, input.DataSourceID, input.TargetID)
	if err != nil {
		uc.l.Errorf(ctx, "execution.usecase.DispatchTarget.GetDispatchContext: %v", err)
		return execution.DispatchTargetOutput{}, uc.mapRepositoryError(err)
	}
	if err := uc.validateDispatchContext(dispatchCtx); err != nil {
		uc.l.Errorf(ctx, "execution.usecase.DispatchTarget.validateDispatchContext: %v", err)
		return execution.DispatchTargetOutput{}, err
	}

	spec, err := uc.buildDispatchSpec(dispatchCtx.Source, dispatchCtx.Target)
	if err != nil {
		uc.l.Errorf(ctx, "execution.usecase.DispatchTarget.buildDispatchSpec: %v", err)
		return execution.DispatchTargetOutput{}, err
	}

	requestedAt := input.RequestedAt
	if requestedAt.IsZero() {
		requestedAt = uc.now()
	}

	scheduledFor := input.ScheduledFor
	if scheduledFor.IsZero() {
		scheduledFor = requestedAt
	}

	triggerType := input.TriggerType
	if triggerType == "" {
		triggerType = model.TriggerTypeManual
	}

	taskID := uuid.NewString()
	requestPayload, err := buildRequestPayload(taskID, spec.Action, spec.Params, requestedAt)
	if err != nil {
		uc.l.Errorf(ctx, "execution.usecase.DispatchTarget.buildRequestPayload: %v", err)
		return execution.DispatchTargetOutput{}, execution.ErrDispatchFailed
	}

	record, err := uc.repo.CreateDispatch(ctx, repo.CreateDispatchOptions{
		Source:         dispatchCtx.Source,
		Target:         dispatchCtx.Target,
		TaskID:         taskID,
		Queue:          spec.Queue,
		Action:         spec.Action,
		TriggerType:    triggerType,
		ScheduledFor:   scheduledFor,
		CronExpr:       input.CronExpr,
		RequestPayload: requestPayload,
		JobPayload:     buildJobPayload(spec.Queue, spec.Action, requestPayload, input),
		CreatedAt:      requestedAt,
	})
	if err != nil {
		uc.l.Errorf(ctx, "execution.usecase.DispatchTarget.CreateDispatch: %v", err)
		return execution.DispatchTargetOutput{}, execution.ErrDispatchFailed
	}

	if err := uc.publishDispatch(ctx, execution.PublishDispatchInput{
		Queue:     spec.Queue,
		TaskID:    taskID,
		Action:    spec.Action,
		Params:    cloneParams(spec.Params),
		CreatedAt: requestedAt,
	}); err != nil {
		uc.l.Errorf(ctx, "execution.usecase.DispatchTarget.publishDispatch: %v", err)
		_ = uc.repo.MarkDispatchFailed(ctx, repo.MarkDispatchFailedOptions{
			SourceID:       dispatchCtx.Source.ID,
			TargetID:       dispatchCtx.Target.ID,
			ExternalTaskID: record.ExternalTask.ID,
			ScheduledJobID: record.ScheduledJob.ID,
			ErrorMessage:   err.Error(),
			FailedAt:       requestedAt,
		})
		return execution.DispatchTargetOutput{}, execution.ErrDispatchFailed
	}

	if err := uc.repo.MarkDispatchPublished(ctx, repo.MarkDispatchPublishedOptions{
		ExternalTaskID: record.ExternalTask.ID,
		ScheduledJobID: record.ScheduledJob.ID,
		PublishedAt:    requestedAt,
	}); err != nil {
		uc.l.Errorf(ctx, "execution.usecase.DispatchTarget.MarkDispatchPublished: %v", err)
		return execution.DispatchTargetOutput{}, execution.ErrDispatchFailed
	}

	return execution.DispatchTargetOutput{
		ScheduledJobID: record.ScheduledJob.ID,
		ExternalTaskID: record.ExternalTask.ID,
		TaskID:         taskID,
		Queue:          spec.Queue,
		Action:         spec.Action,
		Status:         string(model.JobStatusRunning),
	}, nil
}

func (uc *implUseCase) DispatchTargetManually(ctx context.Context, input execution.DispatchTargetManuallyInput) (execution.DispatchTargetManuallyOutput, error) {
	now := uc.now()

	return uc.DispatchTarget(ctx, execution.DispatchTargetInput{
		DataSourceID: input.DataSourceID,
		TargetID:     input.TargetID,
		TriggerType:  model.TriggerTypeManual,
		ScheduledFor: now,
		RequestedAt:  now,
	})
}

func buildRequestPayload(taskID, action string, params map[string]interface{}, createdAt time.Time) (json.RawMessage, error) {
	payload := map[string]interface{}{
		"task_id":    taskID,
		"action":     action,
		"params":     params,
		"created_at": createdAt.Format(time.RFC3339),
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return data, nil
}
