package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
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

	specs, err := uc.buildDispatchSpecs(dispatchCtx.Source, dispatchCtx.Target)
	if err != nil {
		uc.l.Errorf(ctx, "execution.usecase.DispatchTarget.buildDispatchSpecs: %v", err)
		return execution.DispatchTargetOutput{}, err
	}

	return uc.dispatchPrepared(ctx, dispatchCtx, specs, input)
}

func (uc *implUseCase) dispatchPrepared(
	ctx context.Context,
	dispatchCtx repo.DispatchContext,
	specs []execution.DispatchSpec,
	input execution.DispatchTargetInput,
) (execution.DispatchTargetOutput, error) {
	if len(specs) == 0 {
		return execution.DispatchTargetOutput{}, execution.ErrDispatchNotAllowed
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

	scheduledJob, err := uc.repo.CreateScheduledJob(ctx, repo.CreateScheduledJobOptions{
		Source:       dispatchCtx.Source,
		Target:       dispatchCtx.Target,
		TriggerType:  triggerType,
		ScheduledFor: scheduledFor,
		CronExpr:     input.CronExpr,
		JobPayload:   uc.buildJobPayload(specs, input),
		CreatedAt:    requestedAt,
	})
	if err != nil {
		uc.l.Errorf(ctx, "execution.usecase.DispatchTarget.CreateScheduledJob: %v", err)
		if err == repo.ErrDispatchConflict {
			return execution.DispatchTargetOutput{}, execution.ErrDispatchNotAllowed
		}
		return execution.DispatchTargetOutput{}, execution.ErrDispatchFailed
	}

	output := execution.DispatchTargetOutput{
		ScheduledJobID: scheduledJob.ID,
		Status:         string(model.JobStatusRunning),
		TaskCount:      len(specs),
		Tasks:          make([]execution.DispatchTaskOutput, 0, len(specs)),
	}

	var failureMessages []string

	for _, spec := range specs {
		taskOutput, taskErr := uc.dispatchOneSpec(ctx, dispatchCtx, scheduledJob.ID, spec, requestedAt)
		output.Tasks = append(output.Tasks, taskOutput)
		if taskErr != nil {
			output.FailedCount++
			failureMessages = append(failureMessages, taskErr.Error())
			continue
		}
		output.PublishedCount++
	}

	switch {
	case output.PublishedCount == 0:
		completedAt := requestedAt
		finalErrMsg := uc.summarizeDispatchFailures(failureMessages, output.TaskCount)
		if err := uc.repo.FinalizeScheduledJob(ctx, repo.FinalizeScheduledJobOptions{
			ScheduledJobID: scheduledJob.ID,
			SourceID:       dispatchCtx.Source.ID,
			TargetID:       dispatchCtx.Target.ID,
			Status:         model.JobStatusFailed,
			ErrorMessage:   finalErrMsg,
			CompletedAt:    &completedAt,
		}); err != nil {
			uc.l.Errorf(ctx, "execution.usecase.DispatchTarget.FinalizeScheduledJob.failed: %v", err)
		}
		output.Status = string(model.JobStatusFailed)
		return output, execution.ErrDispatchFailed
	case output.FailedCount > 0:
		finalErrMsg := uc.summarizeDispatchFailures(failureMessages, output.TaskCount)
		if err := uc.repo.FinalizeScheduledJob(ctx, repo.FinalizeScheduledJobOptions{
			ScheduledJobID: scheduledJob.ID,
			SourceID:       dispatchCtx.Source.ID,
			TargetID:       dispatchCtx.Target.ID,
			Status:         model.JobStatusPartial,
			ErrorMessage:   finalErrMsg,
		}); err != nil {
			uc.l.Errorf(ctx, "execution.usecase.DispatchTarget.FinalizeScheduledJob.partial: %v", err)
			return execution.DispatchTargetOutput{}, execution.ErrDispatchFailed
		}
		output.Status = string(model.JobStatusPartial)
	default:
		output.Status = string(model.JobStatusRunning)
	}

	return output, nil
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

func (uc *implUseCase) dispatchOneSpec(
	ctx context.Context,
	dispatchCtx repo.DispatchContext,
	scheduledJobID string,
	spec execution.DispatchSpec,
	requestedAt time.Time,
) (execution.DispatchTaskOutput, error) {
	taskID := uuid.NewString()
	requestPayload, err := uc.buildRequestPayload(taskID, spec.Action, spec.Params, requestedAt)
	if err != nil {
		uc.l.Errorf(ctx, "execution.usecase.dispatchOneSpec.buildRequestPayload: %v", err)
		return execution.DispatchTaskOutput{
			TaskID:       taskID,
			Queue:        string(spec.Queue),
			Action:       string(spec.Action),
			Status:       string(model.JobStatusFailed),
			Keyword:      spec.Keyword,
			ErrorMessage: execution.ErrDispatchFailed.Error(),
		}, execution.ErrDispatchFailed
	}

	externalTask, err := uc.repo.CreateExternalTask(ctx, repo.CreateExternalTaskOptions{
		Source:         dispatchCtx.Source,
		Target:         dispatchCtx.Target,
		ScheduledJobID: scheduledJobID,
		TaskID:         taskID,
		Queue:          string(spec.Queue),
		Action:         string(spec.Action),
		RequestPayload: requestPayload,
		CreatedAt:      requestedAt,
	})
	if err != nil {
		uc.l.Errorf(ctx, "execution.usecase.dispatchOneSpec.CreateExternalTask: %v", err)
		return execution.DispatchTaskOutput{
			TaskID:       taskID,
			Queue:        string(spec.Queue),
			Action:       string(spec.Action),
			Status:       string(model.JobStatusFailed),
			Keyword:      spec.Keyword,
			ErrorMessage: execution.ErrDispatchFailed.Error(),
		}, execution.ErrDispatchFailed
	}

	if err := uc.publishDispatch(ctx, execution.PublishDispatchInput{
		Queue:     spec.Queue,
		TaskID:    taskID,
		Action:    spec.Action,
		Params:    uc.cloneParams(spec.Params),
		CreatedAt: requestedAt,
	}); err != nil {
		uc.l.Errorf(ctx, "execution.usecase.dispatchOneSpec.publishDispatch: task_id=%s err=%v", taskID, err)
		markErr := uc.repo.MarkExternalTaskFailed(ctx, repo.MarkExternalTaskFailedOptions{
			ExternalTaskID: externalTask.ID,
			ErrorMessage:   err.Error(),
			FailedAt:       requestedAt,
		})
		if markErr != nil {
			uc.l.Errorf(ctx, "execution.usecase.dispatchOneSpec.MarkExternalTaskFailed: task_id=%s err=%v", taskID, markErr)
		}
		return execution.DispatchTaskOutput{
			ExternalTaskID: externalTask.ID,
			TaskID:         taskID,
			Queue:          string(spec.Queue),
			Action:         string(spec.Action),
			Status:         string(model.JobStatusFailed),
			Keyword:        spec.Keyword,
			ErrorMessage:   err.Error(),
		}, err
	}

	if err := uc.repo.MarkExternalTaskPublished(ctx, repo.MarkExternalTaskPublishedOptions{
		ExternalTaskID: externalTask.ID,
		PublishedAt:    requestedAt,
	}); err != nil {
		uc.l.Errorf(ctx, "execution.usecase.dispatchOneSpec.MarkExternalTaskPublished: task_id=%s err=%v", taskID, err)
		_ = uc.repo.MarkExternalTaskFailed(ctx, repo.MarkExternalTaskFailedOptions{
			ExternalTaskID: externalTask.ID,
			ErrorMessage:   err.Error(),
			FailedAt:       requestedAt,
		})
		return execution.DispatchTaskOutput{
			ExternalTaskID: externalTask.ID,
			TaskID:         taskID,
			Queue:          string(spec.Queue),
			Action:         string(spec.Action),
			Status:         string(model.JobStatusFailed),
			Keyword:        spec.Keyword,
			ErrorMessage:   err.Error(),
		}, err
	}

	uc.l.Infof(ctx, "execution.usecase.dispatchOneSpec.published: source_id=%s target_id=%s scheduled_job_id=%s external_task_id=%s task_id=%s queue=%s action=%s keyword=%s",
		dispatchCtx.Source.ID,
		dispatchCtx.Target.ID,
		scheduledJobID,
		externalTask.ID,
		taskID,
		spec.Queue,
		spec.Action,
		spec.Keyword,
	)

	return execution.DispatchTaskOutput{
		ExternalTaskID: externalTask.ID,
		TaskID:         taskID,
		Queue:          string(spec.Queue),
		Action:         string(spec.Action),
		Status:         string(model.JobStatusRunning),
		Keyword:        spec.Keyword,
	}, nil
}

func (uc *implUseCase) buildRequestPayload(taskID string, action execution.ActionName, params map[string]interface{}, createdAt time.Time) (json.RawMessage, error) {
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

func (uc *implUseCase) summarizeDispatchFailures(messages []string, taskCount int) string {
	unique := make([]string, 0, len(messages))
	seen := make(map[string]struct{}, len(messages))
	for _, message := range messages {
		trimmed := strings.TrimSpace(message)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		unique = append(unique, trimmed)
	}

	if len(unique) == 0 {
		return fmt.Sprintf("failed to publish %d external task(s)", taskCount)
	}

	return fmt.Sprintf("dispatch fan-out encountered %d failure(s): %s", len(messages), strings.Join(unique, "; "))
}
