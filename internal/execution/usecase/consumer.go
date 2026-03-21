package usecase

import (
	"context"
	"strings"
	"time"

	"ingest-srv/internal/execution"
	repo "ingest-srv/internal/execution/repository"
	"ingest-srv/internal/model"
	"ingest-srv/internal/uap"
)

func (uc *implUseCase) HandleCompletion(ctx context.Context, input execution.HandleCompletionInput) error {
	if err := uc.validateCompletionInput(input); err != nil {
		uc.l.Errorf(ctx, "execution.usecase.HandleCompletion: invalid input: %v", err)
		return err
	}

	completionCtx, err := uc.repo.GetCompletionContext(ctx, input.TaskID)
	if err != nil {
		if err == repo.ErrExternalTaskNotFound {
			return execution.ErrCompletionTaskNotFound
		}
		uc.l.Errorf(ctx, "execution.usecase.HandleCompletion: failed to get completion context: %v", err)
		return err
	}
	if completionCtx.ExternalTask.Status == model.JobStatusCancelled && completionCtx.ExternalTask.CompletedAt != nil {
		uc.l.Infof(ctx, "execution.usecase.HandleCompletion: ignore completion for cancelled task_id=%s", input.TaskID)
		return nil
	}

	completedAt := uc.now()
	if parsed, err := time.Parse(time.RFC3339, input.CompletedAt); err == nil {
		completedAt = parsed.UTC()
	}

	switch strings.ToLower(strings.TrimSpace(input.Status)) {
	case "error":
		if uc.isTerminalFailure(completionCtx.ExternalTask.Status) && completionCtx.ExternalTask.CompletedAt != nil {
			uc.l.Infof(ctx, "execution.usecase.HandleCompletion: duplicate error completion task_id=%s", input.TaskID)
			return nil
		}
		errMessage := strings.TrimSpace(input.Error)
		if errMessage == "" {
			errMessage = "crawler returned error completion without error message"
		}
		return uc.repo.CompleteTaskError(ctx, repo.CompleteTaskErrorOptions{
			CompletionContext: completionCtx,
			ErrorMessage:      errMessage,
			CompletedAt:       completedAt,
		})
	case "success":
		exists, err := uc.repo.HasRawBatch(ctx, completionCtx.ExternalTask.SourceID, input.BatchID)
		if err != nil {
			uc.l.Errorf(ctx, "execution.usecase.HandleCompletion: failed to check raw batch existence: %v", err)
			return err
		}
		if exists {
			uc.l.Infof(ctx, "execution.usecase.HandleCompletion: duplicate success completion task_id=%s batch_id=%s", input.TaskID, input.BatchID)
			return nil
		}

		fileInfo, err := uc.verifyMinIOObject(ctx, input.StorageBucket, input.StoragePath)
		if err != nil {
			uc.l.Warnf(ctx, "execution.usecase.HandleCompletion.verifyMinIOObject: task_id=%s err=%v", input.TaskID, err)
			return uc.repo.CompleteTaskError(ctx, repo.CompleteTaskErrorOptions{
				CompletionContext: completionCtx,
				ErrorMessage:      err.Error(),
				CompletedAt:       completedAt,
			})
		}

		var sizeBytes *int64
		if rawSize, ok := input.Metadata["size_bytes"]; ok {
			if parsed := uc.parseInt64(rawSize); parsed != nil {
				sizeBytes = parsed
			}
		}
		if sizeBytes == nil && fileInfo != nil {
			size := fileInfo.Size
			sizeBytes = &size
		}

		rawMetadata, err := uc.marshalMetadata(input.Metadata)
		if err != nil {
			uc.l.Errorf(ctx, "execution.usecase.HandleCompletion.marshalMetadata: %v", err)
			return execution.ErrInvalidCompletionInput
		}

		rawBatch, err := uc.repo.CompleteTaskSuccess(ctx, repo.CompleteTaskSuccessOptions{
			CompletionContext: completionCtx,
			BatchID:           input.BatchID,
			StorageBucket:     input.StorageBucket,
			StoragePath:       input.StoragePath,
			Checksum:          input.Checksum,
			ItemCount:         input.ItemCount,
			SizeBytes:         sizeBytes,
			RawMetadata:       rawMetadata,
			CompletedAt:       completedAt,
		})
		if err == repo.ErrRawBatchAlreadyExists {
			uc.l.Infof(ctx, "execution.usecase.HandleCompletion: duplicate raw batch task_id=%s batch_id=%s", input.TaskID, input.BatchID)
			return nil
		}
		if err != nil {
			return err
		}

		if uc.shouldParseUAP(completionCtx.ExternalTask) {
			parseErr := uc.parser.ParseAndStoreRawBatch(ctx, uap.ParseAndStoreRawBatchInput{
				RawBatchID:     rawBatch.ID,
				ProjectID:      rawBatch.ProjectID,
				SourceID:       rawBatch.SourceID,
				ExternalTaskID: completionCtx.ExternalTask.ID,
				TaskID:         completionCtx.ExternalTask.TaskID,
				Platform:       completionCtx.ExternalTask.Platform,
				Action:         completionCtx.ExternalTask.TaskType,
				StorageBucket:  rawBatch.StorageBucket,
				StoragePath:    rawBatch.StoragePath,
				BatchID:        rawBatch.BatchID,
				RawMetadata:    rawBatch.RawMetadata,
				RequestPayload: completionCtx.ExternalTask.RequestPayload,
				CompletionTime: completedAt,
			})
			if parseErr != nil {
				uc.l.Errorf(ctx, "execution.usecase.HandleCompletion.ParseAndStoreRawBatch: task_id=%s raw_batch_id=%s err=%v", input.TaskID, rawBatch.ID, parseErr)
			}
		}

		return nil
	default:
		uc.l.Errorf(ctx, "execution.usecase.HandleCompletion: invalid status value: %s", input.Status)
		return execution.ErrInvalidCompletionInput
	}
}

func (uc *implUseCase) shouldParseUAP(task model.ExternalTask) bool {
	if uc.parser == nil {
		return false
	}

	return strings.EqualFold(strings.TrimSpace(task.Platform), uap.PlatformTikTok) &&
		strings.EqualFold(strings.TrimSpace(task.TaskType), uap.TaskTypeFullFlow)
}
