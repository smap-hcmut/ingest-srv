package project

import (
	"net/http"
	"strings"
	"time"

	"ingest-srv/pkg/microservice"

	"github.com/smap-hcmut/shared-libs/go/log"
)

func New(l log.Logger, baseURL string, timeoutMS int, internalKey string) microservice.ProjectUseCase {
	timeout := time.Duration(timeoutMS) * time.Millisecond
	if timeoutMS <= 0 {
		timeout = 5 * time.Second
	}

	return &implUseCase{
		l:           l,
		baseURL:     strings.TrimRight(baseURL, "/"),
		internalKey: internalKey,
		client:      &http.Client{Timeout: timeout},
	}
}
