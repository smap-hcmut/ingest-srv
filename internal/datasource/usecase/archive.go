package usecase

import (
	"context"

	"ingest-srv/internal/datasource"
	repo "ingest-srv/internal/datasource/repository"
)

// Archive soft-deletes a data source by ID.
func (uc *implUseCase) Archive(ctx context.Context, id string) error {
	if id == "" {
		uc.l.Warnf(ctx, "datasource.usecase.Archive: empty id")
		return datasource.ErrNotFound
	}

	if err := uc.repo.ArchiveDataSource(ctx, id); err != nil {
		uc.l.Errorf(ctx, "datasource.usecase.Archive.repo.ArchiveDataSource: id=%s err=%v", id, err)
		if err == repo.ErrFailedToGet {
			return datasource.ErrNotFound
		}
		return datasource.ErrDeleteFailed
	}

	return nil
}
