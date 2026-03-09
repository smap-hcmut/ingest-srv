package usecase

import (
	"ingest-srv/internal/datasource"
	repo "ingest-srv/internal/datasource/repository"
	"ingest-srv/pkg/log"
)

type implUseCase struct {
	l    log.Logger
	repo repo.Repository
}

// New creates a new DataSource use case.
func New(l log.Logger, r repo.Repository) datasource.UseCase {
	return &implUseCase{l: l, repo: r}
}
