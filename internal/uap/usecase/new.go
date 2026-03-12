package usecase

import (
	"time"

	"ingest-srv/internal/uap"
	repo "ingest-srv/internal/uap/repository"

	"github.com/smap-hcmut/shared-libs/go/log"
	"github.com/smap-hcmut/shared-libs/go/minio"
)

type implUseCase struct {
	l            log.Logger
	repo         repo.Repository
	minio        minio.MinIO
	outputBucket string
	publisher    uap.Publisher
	uapTopic     string
	now          func() time.Time
}

var _ uap.UseCase = (*implUseCase)(nil)

func New(
	l log.Logger,
	repository repo.Repository,
	minioClient minio.MinIO,
	outputBucket string,
	publisher uap.Publisher,
	uapTopic string,
) uap.UseCase {
	return &implUseCase{
		l:            l,
		repo:         repository,
		minio:        minioClient,
		outputBucket: outputBucket,
		publisher:    publisher,
		uapTopic:     uapTopic,
		now:          func() time.Time { return time.Now().UTC() },
	}
}
