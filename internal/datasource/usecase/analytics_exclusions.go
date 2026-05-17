package usecase

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type hideCrawlTargetRequest struct {
	TargetID     string `json:"target_id"`
	DataSourceID string `json:"data_source_id,omitempty"`
	Reason       string `json:"reason,omitempty"`
	HiddenBy     string `json:"hidden_by,omitempty"`
}

func (uc *implUseCase) hideTargetFromAnalytics(ctx context.Context, dataSourceID, targetID string) error {
	if uc.analyticsBaseURL == "" {
		uc.l.Warnf(ctx, "datasource.usecase.hideTargetFromAnalytics.skip: ANALYTICS_API_INTERNAL_URL is not configured")
		return nil
	}
	if uc.internalKey == "" {
		return fmt.Errorf("internal key is not configured")
	}

	payload := hideCrawlTargetRequest{
		TargetID:     strings.TrimSpace(targetID),
		DataSourceID: strings.TrimSpace(dataSourceID),
		Reason:       "stalker_flush",
		HiddenBy:     "ingest-srv",
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal analytics exclusion payload: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		uc.analyticsBaseURL+"/api/v1/internal/analytics/hidden-crawl-targets",
		bytes.NewReader(body),
	)
	if err != nil {
		return fmt.Errorf("build analytics exclusion request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Internal-Key", uc.internalKey)

	resp, err := uc.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("call analytics exclusion: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("analytics exclusion returned status %d", resp.StatusCode)
	}
	return nil
}
