package usecase

import (
	"context"

	"ingest-srv/internal/datasource"
)

// Detail fetches a single data source by ID.
func (uc *implUseCase) Detail(ctx context.Context, id string) (datasource.DetailOutput, error) {
	if id == "" {
		uc.l.Warnf(ctx, "datasource.usecase.Detail: empty id")
		return datasource.DetailOutput{}, datasource.ErrNotFound
	}

	result, err := uc.repo.DetailDataSource(ctx, id)
	if err != nil {
		uc.l.Errorf(ctx, "datasource.usecase.Detail.repo.DetailDataSource: id=%s err=%v", id, err)
		return datasource.DetailOutput{}, datasource.ErrNotFound
	}

	// Zero value = not found
	if result.ID == "" {
		uc.l.Warnf(ctx, "datasource.usecase.Detail: not found id=%s", id)
		return datasource.DetailOutput{}, datasource.ErrNotFound
	}

	return datasource.DetailOutput{DataSource: result}, nil
}
