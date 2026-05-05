package microservice

import "context"

//go:generate mockery --name ProjectUseCase

type ProjectUseCase interface {
	Detail(ctx context.Context, projectID string) (ProjectDetail, error)
}
