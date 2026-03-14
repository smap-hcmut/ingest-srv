package usecase

import (
	"context"
	"strings"

	dsRepo "ingest-srv/internal/datasource/repository"
	"ingest-srv/internal/dryrun"
	dryrunRepo "ingest-srv/internal/dryrun/repository"
	"ingest-srv/internal/model"

	"github.com/smap-hcmut/shared-libs/go/auth"
)

// Trigger runs one validation-only dryrun for a datasource or grouped crawl target and persists the result.
func (uc *implUseCase) Trigger(ctx context.Context, input dryrun.TriggerInput) (dryrun.TriggerOutput, error) {
	if err := validTriggerInput(input); err != nil {
		uc.l.Warnf(ctx, "dryrun.usecase.Trigger.validTriggerInput: %v", err)
		return dryrun.TriggerOutput{}, err
	}

	source, err := uc.dsRepo.DetailDataSource(ctx, strings.TrimSpace(input.SourceID))
	if err != nil {
		uc.l.Errorf(ctx, "dryrun.usecase.Trigger.dsRepo.DetailDataSource: id=%s err=%v", input.SourceID, err)
		return dryrun.TriggerOutput{}, dryrun.ErrSourceNotFound
	}
	if source.ID == "" {
		return dryrun.TriggerOutput{}, dryrun.ErrSourceNotFound
	}
	switch source.Status {
	case model.SourceStatusPending, model.SourceStatusReady:
	default:
		return dryrun.TriggerOutput{}, dryrun.ErrDryrunNotAllowed
	}

	var target *model.CrawlTarget
	if source.SourceCategory == model.SourceCategoryCrawl {
		if strings.TrimSpace(input.TargetID) == "" {
			return dryrun.TriggerOutput{}, dryrun.ErrTargetRequired
		}
		t, err := uc.dsRepo.GetTarget(ctx, dsRepo.GetTargetOptions{
			DataSourceID: source.ID,
			ID:           strings.TrimSpace(input.TargetID),
		})
		if err != nil {
			uc.l.Errorf(ctx, "dryrun.usecase.Trigger.dsRepo.GetTarget: source=%s target=%s err=%v", input.SourceID, input.TargetID, err)
			return dryrun.TriggerOutput{}, dryrun.ErrTargetNotFound
		}
		if !t.IsActive {
			return dryrun.TriggerOutput{}, dryrun.ErrDryrunNotAllowed
		}
		target = &t
	} else if strings.TrimSpace(input.TargetID) != "" {
		return dryrun.TriggerOutput{}, dryrun.ErrTargetForbidden
	}

	requestedBy, _ := auth.GetUserIDFromContext(ctx)
	running, err := uc.repo.CreateResult(ctx, dryrunRepo.CreateResultOptions{
		SourceID:    source.ID,
		ProjectID:   source.ProjectID,
		TargetID:    strings.TrimSpace(input.TargetID),
		Status:      string(model.DryrunStatusRunning),
		RequestedBy: requestedBy,
		SampleCount: 0,
	})
	if err != nil {
		uc.l.Errorf(ctx, "dryrun.usecase.Trigger.repo.CreateResult: source=%s err=%v", input.SourceID, err)
		return dryrun.TriggerOutput{}, dryrun.ErrCreateFailed
	}

	execResult := uc.exec.Execute(executionInput{
		source: source,
		target: target,
	})
	finalResult, err := uc.repo.UpdateResult(ctx, dryrunRepo.UpdateResultOptions{
		ID:           running.ID,
		Status:       string(execResult.status),
		SampleCount:  0,
		Warnings:     execResult.warnings,
		ErrorMessage: execResult.errorMessage,
	})
	if err != nil {
		uc.l.Errorf(ctx, "dryrun.usecase.Trigger.repo.UpdateResult: id=%s err=%v", running.ID, err)
		return dryrun.TriggerOutput{}, dryrun.ErrUpdateFailed
	}

	updateOpt := dsRepo.UpdateDataSourceOptions{
		ID:                 source.ID,
		DryrunLastResultID: finalResult.ID,
	}
	if execResult.status == model.DryrunStatusWarning {
		updateOpt.Status = string(model.SourceStatusReady)
		updateOpt.DryrunStatus = string(model.DryrunStatusWarning)
	} else {
		updateOpt.Status = string(model.SourceStatusPending)
		updateOpt.DryrunStatus = string(model.DryrunStatusFailed)
	}

	updatedSource, err := uc.dsRepo.UpdateDataSource(ctx, updateOpt)
	if err != nil {
		uc.l.Errorf(ctx, "dryrun.usecase.Trigger.dsRepo.UpdateDataSource: source=%s err=%v", source.ID, err)
		return dryrun.TriggerOutput{}, dryrun.ErrUpdateFailed
	}

	return dryrun.TriggerOutput{
		Result:     finalResult,
		DataSource: updatedSource,
	}, nil
}

// GetLatest returns the latest dryrun result for a source or source-target pair.
func (uc *implUseCase) GetLatest(ctx context.Context, input dryrun.GetLatestInput) (dryrun.GetLatestOutput, error) {
	if err := validGetLatestInput(input); err != nil {
		uc.l.Warnf(ctx, "dryrun.usecase.GetLatest.validGetLatestInput: %v", err)
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
		uc.l.Errorf(ctx, "dryrun.usecase.GetLatest.repo.GetLatest: source=%s target=%s err=%v", input.SourceID, input.TargetID, err)
		return dryrun.GetLatestOutput{}, dryrun.ErrGetFailed
	}

	return dryrun.GetLatestOutput{Result: result}, nil
}

// ListHistory returns paginated dryrun history.
func (uc *implUseCase) ListHistory(ctx context.Context, input dryrun.ListHistoryInput) (dryrun.ListHistoryOutput, error) {
	if err := validListHistoryInput(input); err != nil {
		uc.l.Warnf(ctx, "dryrun.usecase.ListHistory.validListHistoryInput: %v", err)
		return dryrun.ListHistoryOutput{}, err
	}

	input.Paginator.Adjust()
	results, pag, err := uc.repo.ListHistory(ctx, dryrunRepo.ListHistoryOptions{
		SourceID:  strings.TrimSpace(input.SourceID),
		TargetID:  strings.TrimSpace(input.TargetID),
		Paginator: input.Paginator,
	})
	if err != nil {
		uc.l.Errorf(ctx, "dryrun.usecase.ListHistory.repo.ListHistory: source=%s target=%s err=%v", input.SourceID, input.TargetID, err)
		return dryrun.ListHistoryOutput{}, dryrun.ErrListFailed
	}

	return dryrun.ListHistoryOutput{
		Results:   results,
		Paginator: pag,
	}, nil
}
