package datasource

import (
	"context"
)

// UseCase defines the business logic interface for DataSource operations.
type UseCase interface {
	DataSourceUseCase
	ProjectOrchestrationUseCase

	// CrawlTarget sub-resource operations.
	CrawlTargetUseCase
}

// DataSourceUseCase defines CRUD + lifecycle at single-datasource scope.
type DataSourceUseCase interface {
	Create(ctx context.Context, input CreateInput) (CreateOutput, error)
	Detail(ctx context.Context, id string) (DetailOutput, error)
	List(ctx context.Context, input ListInput) (ListOutput, error)
	Update(ctx context.Context, input UpdateInput) (UpdateOutput, error)
	Archive(ctx context.Context, id string) error
	ActivateDataSource(ctx context.Context, id string) (ActivateOutput, error)
	PauseDataSource(ctx context.Context, id string) (PauseOutput, error)
	ResumeDataSource(ctx context.Context, id string) (ResumeOutput, error)
	UpdateCrawlMode(ctx context.Context, input UpdateCrawlModeInput) (UpdateCrawlModeOutput, error)
}

// ProjectOrchestrationUseCase defines project-scope orchestration over datasource runtime.
type ProjectOrchestrationUseCase interface {
	GetActivationReadiness(ctx context.Context, projectID string) (ActivationReadinessOutput, error)
	Activate(ctx context.Context, projectID string) (ProjectLifecycleOutput, error)
	Pause(ctx context.Context, projectID string) (ProjectLifecycleOutput, error)
	Resume(ctx context.Context, projectID string) (ProjectLifecycleOutput, error)
}

type CrawlTargetUseCase interface {
	CreateKeywordTarget(ctx context.Context, input CreateTargetGroupInput) (CreateTargetOutput, error)
	CreateProfileTarget(ctx context.Context, input CreateTargetGroupInput) (CreateTargetOutput, error)
	CreatePostTarget(ctx context.Context, input CreateTargetGroupInput) (CreateTargetOutput, error)
	DetailTarget(ctx context.Context, input DetailTargetInput) (DetailTargetOutput, error)
	ListTargets(ctx context.Context, input ListTargetsInput) (ListTargetsOutput, error)
	UpdateTarget(ctx context.Context, input UpdateTargetInput) (UpdateTargetOutput, error)
	DeleteTarget(ctx context.Context, input DeleteTargetInput) error
}
