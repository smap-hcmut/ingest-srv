package usecase

import (
	"ingest-srv/internal/datasource"
	repo "ingest-srv/internal/datasource/repository"
	"ingest-srv/pkg/microservice"

	"github.com/smap-hcmut/shared-libs/go/log"
)

type implUseCase struct {
	l       log.Logger
	repo    repo.Repository
	project microservice.ProjectUseCase
}

// New creates a new DataSource use case.
func New(l log.Logger, r repo.Repository, project microservice.ProjectUseCase) datasource.UseCase {
	return &implUseCase{l: l, repo: r, project: project}
}
