package execution

import "context"

// UseCase defines execution-plane runtime operations.
type UseCase interface {
	DispatchTarget(ctx context.Context, input DispatchTargetInput) (DispatchTargetOutput, error)
	DispatchTargetManually(ctx context.Context, input DispatchTargetManuallyInput) (DispatchTargetManuallyOutput, error)
	ConsumerUseCase
	CronUseCase
	RuntimeControlUseCase
}

type ConsumerUseCase interface {
	HandleCompletion(ctx context.Context, input HandleCompletionInput) error
}

type CronUseCase interface {
	DispatchDueTargets(ctx context.Context, input DispatchDueTargetsInput) (DispatchDueTargetsOutput, error)
}

type RuntimeControlUseCase interface {
	CancelProjectRuntime(ctx context.Context, input CancelProjectRuntimeInput) error
}
