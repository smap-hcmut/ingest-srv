package usecase

import (
	"time"

	"ingest-srv/internal/execution"
	executionProducer "ingest-srv/internal/execution/delivery/rabbitmq/producer"
	repo "ingest-srv/internal/execution/repository"
	"ingest-srv/internal/uap"
	"ingest-srv/pkg/microservice"

	"github.com/smap-hcmut/shared-libs/go/log"
	"github.com/smap-hcmut/shared-libs/go/minio"
)

type implUseCase struct {
	l         log.Logger
	repo      repo.Repository
	minio     minio.MinIO
	publisher executionProducer.Producer
	parser    uap.UseCase
	project   microservice.ProjectUseCase
	now       func() time.Time
	sleep     func(time.Duration)
}

var _ execution.UseCase = (*implUseCase)(nil)

// New creates a new execution usecase.
func New(l log.Logger, repository repo.Repository, minioClient minio.MinIO, publisher executionProducer.Producer, parser uap.UseCase, project microservice.ProjectUseCase) execution.UseCase {
	return &implUseCase{
		l:         l,
		repo:      repository,
		minio:     minioClient,
		publisher: publisher,
		parser:    parser,
		project:   project,
		now:       func() time.Time { return time.Now().UTC() },
		sleep:     time.Sleep,
	}
}
