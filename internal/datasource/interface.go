package datasource

import (
	"context"
)

// UseCase defines the business logic interface for DataSource operations.
type UseCase interface {
	Create(ctx context.Context, input CreateInput) (CreateOutput, error)
	Detail(ctx context.Context, id string) (DetailOutput, error)
	List(ctx context.Context, input ListInput) (ListOutput, error)
	Update(ctx context.Context, input UpdateInput) (UpdateOutput, error)
	Archive(ctx context.Context, id string) error
	Activate(ctx context.Context, input ActivateInput) (ActivateOutput, error)
	Pause(ctx context.Context, input PauseInput) (PauseOutput, error)
	Resume(ctx context.Context, input ResumeInput) (ResumeOutput, error)
	UpdateCrawlMode(ctx context.Context, input UpdateCrawlModeInput) (UpdateCrawlModeOutput, error)

	// CrawlTarget sub-resource operations.
	CrawlTargetUseCase
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
