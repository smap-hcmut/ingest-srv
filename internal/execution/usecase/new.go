package usecase

import (
	"context"
	"time"

	"ingest-srv/internal/execution"
	repo "ingest-srv/internal/execution/repository"
	"ingest-srv/internal/uap"

	"github.com/smap-hcmut/shared-libs/go/log"
	"github.com/smap-hcmut/shared-libs/go/minio"
)

type taskPublisher interface {
	PublishDispatch(ctx context.Context, input execution.PublishDispatchInput) error
}

type rawBatchParser interface {
	ParseAndStoreRawBatch(ctx context.Context, input uap.ParseAndStoreRawBatchInput) error
}

type implUseCase struct {
	l         log.Logger
	repo      repo.Repository
	minio     minio.MinIO
	publisher taskPublisher
	parser    rawBatchParser
	now       func() time.Time
	sleep     func(time.Duration)
}

var _ execution.UseCase = (*implUseCase)(nil)

// New creates a new execution usecase.
func New(l log.Logger, repository repo.Repository, minioClient minio.MinIO, publisher taskPublisher, parser rawBatchParser) execution.UseCase {
	return &implUseCase{
		l:         l,
		repo:      repository,
		minio:     minioClient,
		publisher: publisher,
		parser:    parser,
		now:       func() time.Time { return time.Now().UTC() },
		sleep:     time.Sleep,
	}
}
