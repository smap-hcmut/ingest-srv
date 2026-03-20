package usecase

import (
	"context"
	"fmt"
	"strings"

	"ingest-srv/internal/dryrun"
	dryrunRepo "ingest-srv/internal/dryrun/repository"
	"ingest-srv/internal/model"
)

// HandleCompletion finalizes one async dryrun result from crawler completion.
func (uc *implUseCase) HandleCompletion(ctx context.Context, input dryrun.HandleCompletionInput) error {
	if err := uc.validateCompletionInput(input); err != nil {
		uc.l.Warnf(ctx, "dryrun.usecase.HandleCompletion.validateCompletionInput: task_id=%s err=%v", input.TaskID, err)
		return err
	}

	result, err := uc.repo.GetByJobID(ctx, strings.TrimSpace(input.TaskID))
	if err != nil {
		if err == dryrunRepo.ErrNotFound {
			return dryrun.ErrCompletionTaskNotFound
		}
		uc.l.Errorf(ctx, "dryrun.usecase.HandleCompletion.repo.GetByJobID: task_id=%s err=%v", input.TaskID, err)
		return dryrun.ErrGetFailed
	}

	if uc.isTerminalDryrunStatus(result.Status) || result.CompletedAt != nil {
		if activateErr := uc.ensureActivatedTargetAfterSuccess(ctx, result); activateErr != nil {
			uc.l.Errorf(ctx, "dryrun.usecase.HandleCompletion.ensureActivatedTargetAfterSuccess.duplicate: result_id=%s err=%v", result.ID, activateErr)
			return activateErr
		}
		uc.l.Infof(ctx, "dryrun.usecase.HandleCompletion: duplicate terminal completion task_id=%s result_id=%s", input.TaskID, result.ID)
		return nil
	}

	if strings.EqualFold(strings.TrimSpace(input.Status), "error") {
		errorMessage := strings.TrimSpace(input.Error)
		if errorMessage == "" {
			errorMessage = "crawler returned error completion without error message"
		}

		finalResult, updateErr := uc.repo.UpdateResult(ctx, dryrunRepo.UpdateResultOptions{
			ID:           result.ID,
			Status:       string(model.DryrunStatusFailed),
			SampleCount:  0,
			CompletedAt:  uc.parseCompletedAt(input.CompletedAt),
			ErrorMessage: errorMessage,
		})
		if updateErr != nil {
			uc.l.Errorf(ctx, "dryrun.usecase.HandleCompletion.repo.UpdateResult.failed: result_id=%s err=%v", result.ID, updateErr)
			return dryrun.ErrUpdateFailed
		}

		if _, applyErr := uc.applyDatasourceResult(ctx, finalResult.SourceID, finalResult.ID, model.DryrunStatusFailed); applyErr != nil {
			return applyErr
		}

		return nil
	}

	rawBytes, err := uc.downloadArtifactBytes(ctx, input.StorageBucket, input.StoragePath)
	if err != nil {
		errorMessage := fmt.Sprintf("download dryrun artifact: %v", err)
		finalResult, updateErr := uc.repo.UpdateResult(ctx, dryrunRepo.UpdateResultOptions{
			ID:           result.ID,
			Status:       string(model.DryrunStatusFailed),
			SampleCount:  0,
			CompletedAt:  uc.parseCompletedAt(input.CompletedAt),
			ErrorMessage: errorMessage,
		})
		if updateErr != nil {
			uc.l.Errorf(ctx, "dryrun.usecase.HandleCompletion.repo.UpdateResult.download_failed: result_id=%s err=%v", result.ID, updateErr)
			return dryrun.ErrUpdateFailed
		}

		if _, applyErr := uc.applyDatasourceResult(ctx, finalResult.SourceID, finalResult.ID, model.DryrunStatusFailed); applyErr != nil {
			return applyErr
		}

		return nil
	}

	updateOpt, finalStatus, err := uc.buildSuccessUpdate(rawBytes, input.ItemCount)
	if err != nil {
		uc.l.Errorf(ctx, "dryrun.usecase.HandleCompletion.buildSuccessUpdate: result_id=%s err=%v", result.ID, err)
		return dryrun.ErrUpdateFailed
	}
	updateOpt.ID = result.ID
	updateOpt.CompletedAt = uc.parseCompletedAt(input.CompletedAt)

	finalResult, err := uc.repo.UpdateResult(ctx, updateOpt)
	if err != nil {
		uc.l.Errorf(ctx, "dryrun.usecase.HandleCompletion.repo.UpdateResult.success: result_id=%s err=%v", result.ID, err)
		return dryrun.ErrUpdateFailed
	}

	if _, applyErr := uc.applyDatasourceResult(ctx, finalResult.SourceID, finalResult.ID, finalStatus); applyErr != nil {
		uc.l.Errorf(ctx, "dryrun.usecase.HandleCompletion.applyDatasourceResult: result_id=%s err=%v", result.ID, applyErr)
		return applyErr
	}

	if activateErr := uc.ensureActivatedTargetAfterSuccess(ctx, finalResult); activateErr != nil {
		uc.l.Errorf(ctx, "dryrun.usecase.HandleCompletion.ensureActivatedTargetAfterSuccess: result_id=%s err=%v", result.ID, activateErr)
		return activateErr
	}

	return nil
}
