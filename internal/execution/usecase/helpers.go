package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"ingest-srv/internal/execution"
	repo "ingest-srv/internal/execution/repository"
	"ingest-srv/internal/model"
	"ingest-srv/pkg/minio"
)

func (uc *implUseCase) validateDispatchContext(ctx repo.DispatchContext) error {
	if ctx.Source.SourceCategory != model.SourceCategoryCrawl {
		return execution.ErrDispatchNotAllowed
	}
	if ctx.Source.Status == model.SourceStatusArchived {
		return execution.ErrDispatchNotAllowed
	}
	if !ctx.Target.IsActive {
		return execution.ErrDispatchNotAllowed
	}
	if ctx.Source.CrawlMode == nil {
		return execution.ErrDispatchNotAllowed
	}
	return nil
}

func (uc *implUseCase) buildDispatchSpec(source model.DataSource, target model.CrawlTarget) (execution.DispatchSpec, error) {
	switch {
	case source.SourceType == model.SourceTypeTikTok && target.TargetType == model.TargetTypeKeyword:
		if len(target.Values) == 0 {
			return execution.DispatchSpec{}, execution.ErrDispatchNotAllowed
		}
		return execution.DispatchSpec{
			Queue:  tikTokTasksQueue,
			Action: actionFullFlow,
			Params: map[string]interface{}{
				"keywords": target.Values,
			},
		}, nil
	case source.SourceType == model.SourceTypeFacebook && target.TargetType == model.TargetTypePostURL:
		parseIDs, err := parseFacebookParseIDs(target.PlatformMeta)
		if err != nil {
			return execution.DispatchSpec{}, err
		}
		return execution.DispatchSpec{
			Queue:  facebookTasksQueue,
			Action: actionPostDetail,
			Params: map[string]interface{}{
				"parse_ids": parseIDs,
			},
		}, nil
	default:
		return execution.DispatchSpec{}, execution.ErrUnsupportedDispatchMapping
	}
}

func (uc *implUseCase) publishDispatch(ctx context.Context, input execution.PublishDispatchInput) error {
	if uc.publisher == nil {
		return fmt.Errorf("execution publisher is not initialized")
	}
	return uc.publisher.PublishDispatch(ctx, input)
}

func (uc *implUseCase) verifyMinIOObject(ctx context.Context, bucket, path string) (*minio.FileInfo, error) {
	if uc.minio == nil {
		return nil, fmt.Errorf("minio client is not initialized")
	}

	var lastErr error
	for attempt := 0; attempt < minioVerifyRetryAttempts; attempt++ {
		exists, err := uc.minio.FileExists(ctx, bucket, path)
		if err == nil && exists {
			info, statErr := uc.minio.GetFileInfo(ctx, bucket, path)
			if statErr == nil {
				return info, nil
			}
			lastErr = statErr
		} else if err != nil {
			lastErr = err
		} else {
			lastErr = fmt.Errorf("minio object missing bucket=%s path=%s", bucket, path)
		}
		if attempt < minioVerifyRetryAttempts-1 {
			uc.sleep(200 * time.Millisecond)
		}
	}

	return nil, lastErr
}

func (uc *implUseCase) mapRepositoryError(err error) error {
	switch err {
	case repo.ErrDataSourceNotFound:
		return execution.ErrDataSourceNotFound
	case repo.ErrTargetNotFound:
		return execution.ErrTargetNotFound
	default:
		return execution.ErrDispatchFailed
	}
}

func validateCompletionInput(input execution.HandleCompletionInput) error {
	if strings.TrimSpace(input.TaskID) == "" {
		return execution.ErrInvalidCompletionInput
	}
	switch strings.ToLower(strings.TrimSpace(input.Status)) {
	case "success":
		if strings.TrimSpace(input.StorageBucket) == "" || strings.TrimSpace(input.StoragePath) == "" || strings.TrimSpace(input.BatchID) == "" {
			return execution.ErrInvalidCompletionInput
		}
	case "error":
	default:
		return execution.ErrInvalidCompletionInput
	}
	return nil
}

func parseFacebookParseIDs(platformMeta json.RawMessage) ([]string, error) {
	if len(platformMeta) == 0 {
		return nil, execution.ErrPlatformMetaParseIDs
	}

	var payload struct {
		ParseIDs []string `json:"parse_ids"`
	}
	if err := json.Unmarshal(platformMeta, &payload); err != nil {
		return nil, execution.ErrPlatformMetaParseIDs
	}
	if len(payload.ParseIDs) == 0 {
		return nil, execution.ErrPlatformMetaParseIDs
	}
	return payload.ParseIDs, nil
}

func buildJobPayload(queue, action string, requestPayload json.RawMessage, input execution.DispatchTargetInput) json.RawMessage {
	payload := map[string]interface{}{
		"queue":           queue,
		"action":          action,
		"request_payload": json.RawMessage(requestPayload),
	}
	if input.TriggerType != "" {
		payload["trigger_type"] = input.TriggerType
	}
	if !input.ScheduledFor.IsZero() {
		payload["scheduled_for"] = input.ScheduledFor.Format(time.RFC3339)
	}
	if !input.RequestedAt.IsZero() {
		payload["requested_at"] = input.RequestedAt.Format(time.RFC3339)
	}
	if strings.TrimSpace(input.CronExpr) != "" {
		payload["cron_expr"] = strings.TrimSpace(input.CronExpr)
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return nil
	}
	return data
}

func cloneParams(input map[string]interface{}) map[string]interface{} {
	if len(input) == 0 {
		return nil
	}

	out := make(map[string]interface{}, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}

func parseInt64(v interface{}) *int64 {
	switch val := v.(type) {
	case int:
		parsed := int64(val)
		return &parsed
	case int32:
		parsed := int64(val)
		return &parsed
	case int64:
		return &val
	case float64:
		parsed := int64(val)
		return &parsed
	case json.Number:
		if parsed, err := val.Int64(); err == nil {
			return &parsed
		}
	}
	return nil
}

func marshalMetadata(metadata map[string]interface{}) (json.RawMessage, error) {
	if len(metadata) == 0 {
		return nil, nil
	}
	data, err := json.Marshal(metadata)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func isTerminalFailure(status model.JobStatus) bool {
	return status == model.JobStatusFailed || status == model.JobStatusCancelled
}

func computeEffectiveInterval(source model.DataSource, target model.CrawlTarget) (time.Duration, error) {
	if source.CrawlMode == nil {
		return 0, fmt.Errorf("crawl mode is required")
	}
	if target.CrawlIntervalMinutes <= 0 {
		return 0, fmt.Errorf("crawl interval minutes must be greater than 0")
	}

	multiplier, err := getModeMultiplier(*source.CrawlMode)
	if err != nil {
		return 0, err
	}

	effectiveMinutes := int(math.Round(float64(target.CrawlIntervalMinutes) * multiplier))
	if effectiveMinutes < defaultMinIntervalMinute {
		effectiveMinutes = defaultMinIntervalMinute
	}
	if effectiveMinutes > defaultMaxIntervalMinute {
		effectiveMinutes = defaultMaxIntervalMinute
	}

	return time.Duration(effectiveMinutes) * time.Minute, nil
}

func getModeMultiplier(mode model.CrawlMode) (float64, error) {
	switch mode {
	case model.CrawlModeNormal:
		return normalModeMultiplier, nil
	case model.CrawlModeCrisis:
		return crisisModeMultiplier, nil
	case model.CrawlModeSleep:
		return sleepModeMultiplier, nil
	default:
		return 0, fmt.Errorf("unsupported crawl mode %s", mode)
	}
}
