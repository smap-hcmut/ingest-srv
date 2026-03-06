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

	// CrawlTarget sub-resource operations.
	CreateTarget(ctx context.Context, input CreateTargetInput) (CreateTargetOutput, error)
	DetailTarget(ctx context.Context, id string) (DetailTargetOutput, error)
	ListTargets(ctx context.Context, input ListTargetsInput) (ListTargetsOutput, error)
	UpdateTarget(ctx context.Context, input UpdateTargetInput) (UpdateTargetOutput, error)
	DeleteTarget(ctx context.Context, id string) error
}
