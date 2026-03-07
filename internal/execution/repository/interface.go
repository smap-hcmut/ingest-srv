package repository

import (
	"context"
	"time"
)

// Repository defines persistence operations for the execution runtime.
type Repository interface {
	GetDispatchContext(ctx context.Context, dataSourceID, targetID string) (DispatchContext, error)
	ListDueTargets(ctx context.Context, now time.Time, limit int) ([]DueTarget, error)
	ClaimTarget(ctx context.Context, opt ClaimTargetOptions) (bool, error)
	CreateDispatch(ctx context.Context, opt CreateDispatchOptions) (DispatchRecord, error)
	MarkDispatchPublished(ctx context.Context, opt MarkDispatchPublishedOptions) error
	MarkDispatchFailed(ctx context.Context, opt MarkDispatchFailedOptions) error
	GetCompletionContext(ctx context.Context, taskID string) (CompletionContext, error)
	HasRawBatch(ctx context.Context, sourceID, batchID string) (bool, error)
	CompleteTaskSuccess(ctx context.Context, opt CompleteTaskSuccessOptions) error
	CompleteTaskError(ctx context.Context, opt CompleteTaskErrorOptions) error
}
