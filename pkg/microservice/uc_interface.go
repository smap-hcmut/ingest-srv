package microservice

import "context"

type ProjectUseCase interface {
	Detail(ctx context.Context, projectID string) (ProjectDetail, error)
}
