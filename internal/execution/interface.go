package execution

import "context"

// UseCase defines execution-plane runtime operations.
type UseCase interface {
	DispatchTarget(ctx context.Context, input DispatchTargetInput) (DispatchTargetOutput, error)
	DispatchTargetManually(ctx context.Context, input DispatchTargetManuallyInput) (DispatchTargetManuallyOutput, error)
	ConsumerUseCase
	ProducerUseCase
	CronUseCase
}

type ConsumerUseCase interface {
	HandleCompletion(ctx context.Context, input HandleCompletionInput) error
}

type ProducerUseCase interface {
	DispatchDueTargets(ctx context.Context, input DispatchDueTargetsInput) (DispatchDueTargetsOutput, error)
}

type CronUseCase interface {
	DispatchDueTargets(ctx context.Context, input DispatchDueTargetsInput) (DispatchDueTargetsOutput, error)
}