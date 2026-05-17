package usecase

import (
	"net/http"
	"os"
	"strings"
	"time"

	"ingest-srv/internal/datasource"
	repo "ingest-srv/internal/datasource/repository"
	"ingest-srv/internal/execution"
	"ingest-srv/pkg/microservice"

	"github.com/smap-hcmut/shared-libs/go/log"
)

type implUseCase struct {
	l       log.Logger
	repo    repo.Repository
	project microservice.ProjectUseCase
	exec    execution.RuntimeControlUseCase

	analyticsBaseURL string
	internalKey      string
	httpClient       *http.Client
}

// New creates a new DataSource use case.
func New(l log.Logger, r repo.Repository, project microservice.ProjectUseCase, exec execution.RuntimeControlUseCase) datasource.UseCase {
	return &implUseCase{
		l:                l,
		repo:             r,
		project:          project,
		exec:             exec,
		analyticsBaseURL: strings.TrimRight(strings.TrimSpace(os.Getenv("ANALYTICS_API_INTERNAL_URL")), "/"),
		internalKey:      strings.TrimSpace(os.Getenv("INTERNAL_KEY")),
		httpClient:       &http.Client{Timeout: 5 * time.Second},
	}
}
