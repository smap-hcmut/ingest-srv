package usecase

import (
	dsRepo "ingest-srv/internal/datasource/repository"
	"ingest-srv/internal/dryrun"
	dryrunRepo "ingest-srv/internal/dryrun/repository"
	"ingest-srv/pkg/log"
)

type executor interface {
	Execute(input executionInput) executionResult
}

type implUseCase struct {
	l      log.Logger
	repo   dryrunRepo.Repository
	dsRepo dsRepo.Repository
	exec   executor
}

// New creates a dryrun usecase with the local validation executor.
func New(l log.Logger, repo dryrunRepo.Repository, dsRepo dsRepo.Repository) dryrun.UseCase {
	return &implUseCase{
		l:      l,
		repo:   repo,
		dsRepo: dsRepo,
		exec:   localExecutor{},
	}
}
