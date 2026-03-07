package usecase

import (
	"context"
	"time"

	"ingest-srv/internal/execution"
	repo "ingest-srv/internal/execution/repository"
	"ingest-srv/pkg/log"
	"ingest-srv/pkg/minio"
)

type taskPublisher interface {
	PublishDispatch(ctx context.Context, input execution.PublishDispatchInput) error
}

type implUseCase struct {
	l         log.Logger
	repo      repo.Repository
	minio     minio.MinIO
	publisher taskPublisher
	now       func() time.Time
	sleep     func(time.Duration)
}

var _ execution.UseCase = (*implUseCase)(nil)

// New creates a new execution usecase.
func New(l log.Logger, repository repo.Repository, minioClient minio.MinIO, publisher taskPublisher) execution.UseCase {
	return &implUseCase{
		l:         l,
		repo:      repository,
		minio:     minioClient,
		publisher: publisher,
		now:       func() time.Time { return time.Now().UTC() },
		sleep:     time.Sleep,
	}
}
