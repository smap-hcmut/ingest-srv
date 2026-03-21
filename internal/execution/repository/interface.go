package repository

import (
	"context"
	"time"

	"ingest-srv/internal/model"
)

// Repository defines persistence operations for the execution runtime.
type Repository interface {
	GetDispatchContext(ctx context.Context, dataSourceID, targetID string) (DispatchContext, error)
	DispatchRepository
	CompletionRepository
	DueTargetRepository
	RuntimeControlRepository
}
type DispatchRepository interface {
	CreateScheduledJob(ctx context.Context, opt CreateScheduledJobOptions) (model.ScheduledJob, error)
	CreateExternalTask(ctx context.Context, opt CreateExternalTaskOptions) (model.ExternalTask, error)
	MarkExternalTaskPublished(ctx context.Context, opt MarkExternalTaskPublishedOptions) error
	MarkExternalTaskFailed(ctx context.Context, opt MarkExternalTaskFailedOptions) error
	FinalizeScheduledJob(ctx context.Context, opt FinalizeScheduledJobOptions) error
}

type CompletionRepository interface {
	GetCompletionContext(ctx context.Context, taskID string) (CompletionContext, error)
	HasRawBatch(ctx context.Context, sourceID, batchID string) (bool, error)
	CompleteTaskSuccess(ctx context.Context, opt CompleteTaskSuccessOptions) (model.RawBatch, error)
	CompleteTaskError(ctx context.Context, opt CompleteTaskErrorOptions) error
}

type DueTargetRepository interface {
	ListDueTargets(ctx context.Context, now time.Time, limit int) ([]DueTarget, error)
	ClaimTarget(ctx context.Context, opt ClaimTargetOptions) (bool, error)
	ReleaseClaimTarget(ctx context.Context, opt ReleaseClaimTargetOptions) error
}

type RuntimeControlRepository interface {
	CancelProjectRuntime(ctx context.Context, opt CancelProjectRuntimeOptions) error
}
