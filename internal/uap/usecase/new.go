package usecase

import (
	"net/http"
	"time"

	"ingest-srv/internal/uap"
	repo "ingest-srv/internal/uap/repository"

	"github.com/smap-hcmut/shared-libs/go/log"
	"github.com/smap-hcmut/shared-libs/go/minio"
)

type implUseCase struct {
	l                  log.Logger
	repo               repo.Repository
	minio              minio.MinIO
	outputBucket       string
	publisher          uap.Publisher
	publishTopic       string
	parsers            map[parseKey]parseFunc
	subtitleHTTPClient *http.Client
	now                func() time.Time
}

var _ uap.UseCase = (*implUseCase)(nil)

func New(
	l log.Logger,
	repository repo.Repository,
	minioClient minio.MinIO,
	outputBucket string,
	publisher uap.Publisher,
) uap.UseCase {
	uc := &implUseCase{
		l:                  l,
		repo:               repository,
		minio:              minioClient,
		outputBucket:       outputBucket,
		publisher:          publisher,
		subtitleHTTPClient: &http.Client{Timeout: 10 * time.Second},
		now:                func() time.Time { return time.Now().UTC() },
	}
	if publisher != nil {
		uc.publishTopic = publisher.Topic()
	}
	uc.parsers = uc.buildParseRegistry()
	return uc
}
