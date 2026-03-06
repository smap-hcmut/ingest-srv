package usecase

import (
	"context"

	"ingest-srv/internal/datasource"
	repo "ingest-srv/internal/datasource/repository"
)

// ListTargets returns all crawl targets for a data source.
func (uc *implUseCase) ListTargets(ctx context.Context, input datasource.ListTargetsInput) (datasource.ListTargetsOutput, error) {
	if input.DataSourceID == "" {
		return datasource.ListTargetsOutput{}, datasource.ErrProjectIDRequired
	}

	opt := repo.ListTargetsOptions{
		DataSourceID: input.DataSourceID,
		TargetType:   input.TargetType,
		IsActive:     input.IsActive,
	}

	results, err := uc.repo.ListTargets(ctx, opt)
	if err != nil {
		uc.l.Errorf(ctx, "datasource.usecase.ListTargets.repo.ListTargets: %v", err)
		return datasource.ListTargetsOutput{}, datasource.ErrTargetListFailed
	}

	return datasource.ListTargetsOutput{Targets: results}, nil
}
