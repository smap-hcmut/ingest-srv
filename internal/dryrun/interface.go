package dryrun

import "context"

// UseCase defines dryrun control-plane operations.
type UseCase interface {
	Trigger(ctx context.Context, input TriggerInput) (TriggerOutput, error)
	GetLatest(ctx context.Context, input GetLatestInput) (GetLatestOutput, error)
	ListHistory(ctx context.Context, input ListHistoryInput) (ListHistoryOutput, error)
}
