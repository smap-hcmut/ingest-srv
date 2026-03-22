package usecase

import (
	"time"

	"ingest-srv/internal/datasource"
	"ingest-srv/internal/dryrun"
	dryrunProducer "ingest-srv/internal/dryrun/delivery/rabbitmq/producer"
	dryrunRepo "ingest-srv/internal/dryrun/repository"

	"github.com/smap-hcmut/shared-libs/go/log"
	"github.com/smap-hcmut/shared-libs/go/minio"
)

type implUseCase struct {
	l         log.Logger
	repo      dryrunRepo.Repository
	dsUC      datasource.UseCase
	minio     minio.MinIO
	publisher dryrunProducer.Producer
	now       func() time.Time
}

var _ dryrun.UseCase = (*implUseCase)(nil)

// New creates a new dryrun usecase.
func New(l log.Logger, repo dryrunRepo.Repository, dsUC datasource.UseCase, minioClient minio.MinIO, publisher dryrunProducer.Producer) dryrun.UseCase {
	return &implUseCase{
		l:         l,
		repo:      repo,
		dsUC:      dsUC,
		minio:     minioClient,
		publisher: publisher,
		now:       func() time.Time { return time.Now().UTC() },
	}
}
