package usecase

import (
	"context"

	"ingest-srv/internal/datasource"
)

// DetailTarget fetches a single crawl target by ID.
func (uc *implUseCase) DetailTarget(ctx context.Context, id string) (datasource.DetailTargetOutput, error) {
	result, err := uc.repo.GetTarget(ctx, id)
	if err != nil {
		uc.l.Errorf(ctx, "datasource.usecase.DetailTarget.repo.GetTarget: id=%s err=%v", id, err)
		return datasource.DetailTargetOutput{}, datasource.ErrTargetNotFound
	}

	return datasource.DetailTargetOutput{Target: result}, nil
}
