package project

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"ingest-srv/pkg/microservice"
)

func (uc *implUseCase) Detail(ctx context.Context, projectID string) (microservice.ProjectDetail, error) {
	trimmedProjectID := strings.TrimSpace(projectID)
	endpoint := uc.buildEndpoint(fmt.Sprintf("/projects/%s", url.PathEscape(trimmedProjectID)))
	body, status, err := uc.doRequest(ctx, http.MethodGet, endpoint)
	if err != nil {
		return microservice.ProjectDetail{}, err
	}
	if status != http.StatusOK {
		return microservice.ProjectDetail{}, mapStatusError(status, body)
	}

	var envelope responseEnvelope
	if err := json.Unmarshal(body, &envelope); err != nil {
		return microservice.ProjectDetail{}, fmt.Errorf("%w: unmarshal project detail envelope: %v", microservice.ErrRequestFailed, err)
	}

	var dto projectRespDTO
	if err := json.Unmarshal(envelope.Data, &dto); err != nil {
		return microservice.ProjectDetail{}, fmt.Errorf("%w: unmarshal project detail data: %v", microservice.ErrRequestFailed, err)
	}

	return microservice.ProjectDetail{
		ID:     dto.Project.ID,
		Status: microservice.ProjectStatus(strings.TrimSpace(dto.Project.Status)),
	}, nil
}

func (uc *implUseCase) doRequest(ctx context.Context, method, endpoint string) ([]byte, int, error) {
	if method != http.MethodGet {
		return nil, 0, fmt.Errorf("%w: unsupported method=%s", microservice.ErrRequestFailed, method)
	}

	req, err := http.NewRequestWithContext(ctx, method, endpoint, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("%w: build request: %v", microservice.ErrRequestFailed, err)
	}
	req.Header.Set("Accept", "application/json")
	if uc.internalKey != "" {
		req.Header.Set(internalAuthHeader, uc.internalKey)
	}

	resp, err := uc.client.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("%w: %v", microservice.ErrRequestFailed, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("%w: read response body: %v", microservice.ErrRequestFailed, err)
	}

	return body, resp.StatusCode, nil
}

func mapStatusError(status int, body []byte) error {
	trimmedBody := strings.TrimSpace(string(body))
	switch status {
	case http.StatusBadRequest:
		return fmt.Errorf("%w: %s", microservice.ErrBadRequest, trimmedBody)
	case http.StatusUnauthorized:
		return fmt.Errorf("%w: %s", microservice.ErrUnauthorized, trimmedBody)
	case http.StatusForbidden:
		return fmt.Errorf("%w: %s", microservice.ErrForbidden, trimmedBody)
	default:
		return fmt.Errorf("%w: status=%d body=%s", microservice.ErrRequestFailed, status, trimmedBody)
	}
}
