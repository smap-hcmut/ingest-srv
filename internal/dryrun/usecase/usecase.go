package usecase

import (
	"context"
	"strings"

	"ingest-srv/internal/dryrun"
	dryrunRepo "ingest-srv/internal/dryrun/repository"
	"ingest-srv/internal/model"

	"github.com/google/uuid"
	"github.com/smap-hcmut/shared-libs/go/auth"
)

// Trigger runs one async dryrun for a datasource or grouped crawl target.
func (uc *implUseCase) Trigger(ctx context.Context, input dryrun.TriggerInput) (dryrun.TriggerOutput, error) {
	if err := uc.validateTriggerInput(input); err != nil {
		uc.l.Warnf(ctx, "dryrun.usecase.Trigger.validateTriggerInput: %v", err)
		return dryrun.TriggerOutput{}, err
	}

	source, err := uc.getSource(ctx, input.SourceID)
	if err != nil {
		uc.l.Errorf(ctx, "dryrun.usecase.Trigger.getSource: source_id=%s err=%v", input.SourceID, err)
		return dryrun.TriggerOutput{}, err
	}

	switch source.Status {
	case model.SourceStatusPending, model.SourceStatusReady:
	default:
		uc.l.Warnf(ctx, "dryrun.usecase.Trigger.invalidSourceStatus: source_id=%s status=%s", source.ID, source.Status)
		return dryrun.TriggerOutput{}, dryrun.ErrDryrunNotAllowed
	}

	var target *model.CrawlTarget
	if source.SourceCategory == model.SourceCategoryCrawl {
		if strings.TrimSpace(input.TargetID) == "" {
			uc.l.Warnf(ctx, "dryrun.usecase.Trigger.missingTargetID: source_id=%s", source.ID)
			return dryrun.TriggerOutput{}, dryrun.ErrTargetRequired
		}

		resolvedTarget, err := uc.getTarget(ctx, source.ID, input.TargetID)
		if err != nil {
			uc.l.Errorf(ctx, "dryrun.usecase.Trigger.getTarget: source_id=%s target_id=%s err=%v", source.ID, input.TargetID, err)
			return dryrun.TriggerOutput{}, err
		}
		target = &resolvedTarget
	} else if strings.TrimSpace(input.TargetID) != "" {
		uc.l.Warnf(ctx, "dryrun.usecase.Trigger.targetIDNotAllowed: source_id=%s target_id=%s", source.ID, input.TargetID)
		return dryrun.TriggerOutput{}, dryrun.ErrTargetForbidden
	}

	sampleLimit := uc.normalizeSampleLimit(input.SampleLimit)
	spec, _, err := uc.buildDispatchSpec(source, target, sampleLimit)
	if err != nil {
		uc.l.Errorf(ctx, "dryrun.usecase.Trigger.buildDispatchSpec: source_id=%s target_id=%s err=%v", source.ID, input.TargetID, err)
		return dryrun.TriggerOutput{}, err
	}

	taskID := uuid.NewString()
	requestedBy, _ := auth.GetUserIDFromContext(ctx)
	running, err := uc.repo.CreateResult(ctx, dryrunRepo.CreateResultOptions{
		SourceID:    source.ID,
		ProjectID:   source.ProjectID,
		TargetID:    strings.TrimSpace(input.TargetID),
		JobID:       taskID,
		Status:      string(model.DryrunStatusRunning),
		RequestedBy: requestedBy,
		SampleCount: 0,
	})
	if err != nil {
		uc.l.Errorf(ctx, "dryrun.usecase.Trigger.repo.CreateResult: source_id=%s err=%v", source.ID, err)
		return dryrun.TriggerOutput{}, dryrun.ErrCreateFailed
	}

	updatedSource, err := uc.markDatasourceRunning(ctx, source.ID, running.ID)
	if err != nil {
		return dryrun.TriggerOutput{}, err
	}

	if err := uc.publishDispatch(ctx, dryrun.PublishDispatchInput{
		Queue:     spec.Queue,
		TaskID:    taskID,
		Action:    spec.Action,
		Params:    spec.Params,
		CreatedAt: uc.now(),
	}); err != nil {
		finalResult, finalSource, failErr := uc.failDispatch(ctx, running, err.Error())
		if failErr != nil {
			return dryrun.TriggerOutput{}, failErr
		}
		return dryrun.TriggerOutput{
			Result:     finalResult,
			DataSource: finalSource,
		}, dryrun.ErrDispatchFailed
	}

	return dryrun.TriggerOutput{
		Result:     running,
		DataSource: updatedSource,
	}, nil
}

// GetLatest returns the latest dryrun result for a source or source-target pair.
func (uc *implUseCase) GetLatest(ctx context.Context, input dryrun.GetLatestInput) (dryrun.GetLatestOutput, error) {
	if err := uc.validateGetLatestInput(input); err != nil {
		uc.l.Warnf(ctx, "dryrun.usecase.GetLatest.validateGetLatestInput: %v", err)
		return dryrun.GetLatestOutput{}, err
	}

	result, err := uc.repo.GetLatest(ctx, dryrunRepo.GetLatestOptions{
		SourceID: strings.TrimSpace(input.SourceID),
		TargetID: strings.TrimSpace(input.TargetID),
	})
	if err != nil {
		if err == dryrunRepo.ErrNotFound {
			return dryrun.GetLatestOutput{}, dryrun.ErrResultNotFound
		}
		uc.l.Errorf(ctx, "dryrun.usecase.GetLatest.repo.GetLatest: source_id=%s target_id=%s err=%v", input.SourceID, input.TargetID, err)
		return dryrun.GetLatestOutput{}, dryrun.ErrGetFailed
	}

	return dryrun.GetLatestOutput{Result: result}, nil
}

// ListHistory returns paginated dryrun history.
func (uc *implUseCase) ListHistory(ctx context.Context, input dryrun.ListHistoryInput) (dryrun.ListHistoryOutput, error) {
	if err := uc.validateListHistoryInput(input); err != nil {
		uc.l.Warnf(ctx, "dryrun.usecase.ListHistory.validateListHistoryInput: %v", err)
		return dryrun.ListHistoryOutput{}, err
	}

	input.Paginator.Adjust()
	results, pag, err := uc.repo.ListHistory(ctx, dryrunRepo.ListHistoryOptions{
		SourceID:  strings.TrimSpace(input.SourceID),
		TargetID:  strings.TrimSpace(input.TargetID),
		Paginator: input.Paginator,
	})
	if err != nil {
		uc.l.Errorf(ctx, "dryrun.usecase.ListHistory.repo.ListHistory: source_id=%s target_id=%s err=%v", input.SourceID, input.TargetID, err)
		return dryrun.ListHistoryOutput{}, dryrun.ErrListFailed
	}

	return dryrun.ListHistoryOutput{
		Results:   results,
		Paginator: pag,
	}, nil
}
