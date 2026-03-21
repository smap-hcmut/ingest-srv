package project

import (
	"encoding/json"
	"ingest-srv/pkg/microservice"

	pkghttp "github.com/smap-hcmut/shared-libs/go/httpclient"
	"github.com/smap-hcmut/shared-libs/go/log"
)

type implUseCase struct {
	l           log.Logger
	baseURL     string
	internalKey string
	client      *pkghttp.TracedHTTPClient
}

var _ microservice.ProjectUseCase = (*implUseCase)(nil)

type responseEnvelope struct {
	ErrorCode int             `json:"error_code"`
	Message   string          `json:"message"`
	Data      json.RawMessage `json:"data"`
}

type projectRespDTO struct {
	Project projectDetailDTO `json:"project"`
}

type projectDetailDTO struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}
