package usecase

import (
	"context"

	"ingest-srv/internal/datasource"
)

// DeleteTarget hard-deletes a crawl target by ID.
func (uc *implUseCase) DeleteTarget(ctx context.Context, id string) error {
	if id == "" {
		return datasource.ErrTargetNotFound
	}

	if err := uc.repo.DeleteTarget(ctx, id); err != nil {
		uc.l.Errorf(ctx, "datasource.usecase.DeleteTarget.repo.DeleteTarget: id=%s err=%v", id, err)
		return datasource.ErrTargetDeleteFailed
	}

	return nil
}
