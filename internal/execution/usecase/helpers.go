package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"ingest-srv/internal/execution"
	executionRabbit "ingest-srv/internal/execution/delivery/rabbitmq"
	repo "ingest-srv/internal/execution/repository"
	"ingest-srv/internal/model"

	"github.com/smap-hcmut/shared-libs/go/minio"
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

func (uc *implUseCase) validateScheduledDispatchContext(ctx repo.DispatchContext) error {
	if err := uc.validateDispatchContext(ctx); err != nil {
		return err
	}
	if ctx.Source.Status != model.SourceStatusActive {
		return execution.ErrDispatchNotAllowed
	}
	return nil
}

func (uc *implUseCase) buildDispatchSpecs(source model.DataSource, target model.CrawlTarget) ([]execution.DispatchSpec, error) {
	switch {
	case source.SourceType == model.SourceTypeTikTok && target.TargetType == model.TargetTypeKeyword:
		keywords := uc.extractKeywords(target.Values)
		if len(keywords) == 0 {
			uc.l.Warnf(context.Background(), "execution.usecase.buildDispatchSpecs.tiktokFullFlowInvalidKeywordTarget: target_id=%s keyword_count=%d", target.ID, len(target.Values))
			return nil, execution.ErrDispatchNotAllowed
		}
		specs := make([]execution.DispatchSpec, 0, len(keywords))
		for _, keyword := range keywords {
			specs = append(specs, execution.DispatchSpec{
				Queue:   execution.QueueName(executionRabbit.TikTokTasksQueueName),
				Action:  execution.ActionNameFullFlow,
				Keyword: keyword,
				Params: map[string]interface{}{
					"keyword":       keyword,
					"limit":         execution.TikTokFullFlowLimit,
					"threshold":     execution.TikTokFullFlowThreshold,
					"comment_count": execution.TikTokFullFlowCommentCount,
				},
			})
		}
		return specs, nil
	case source.SourceType == model.SourceTypeFacebook && target.TargetType == model.TargetTypePostURL:
		parseIDs, err := uc.parseFacebookParseIDs(target.PlatformMeta)
		if err != nil {
			return nil, err
		}
		return []execution.DispatchSpec{{
			Queue:  execution.QueueName(executionRabbit.FacebookTasksQueueName),
			Action: execution.ActionNamePostDetail,
			Params: map[string]interface{}{
				"parse_ids": parseIDs,
			},
		}}, nil
	default:
		return nil, execution.ErrUnsupportedDispatchMapping
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
	for attempt := 0; attempt < execution.MinioVerifyRetryAttempts; attempt++ {
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
		if attempt < execution.MinioVerifyRetryAttempts-1 {
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

func (uc *implUseCase) validateCompletionInput(input execution.HandleCompletionInput) error {
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

func (uc *implUseCase) parseFacebookParseIDs(platformMeta json.RawMessage) ([]string, error) {
	if len(platformMeta) == 0 {
		return nil, execution.ErrPlatformMetaParseIDs
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(platformMeta, &payload); err != nil {
		return nil, execution.ErrPlatformMetaParseIDs
	}

	rawParseIDs, ok := payload["parse_ids"].([]interface{})
	if !ok || len(rawParseIDs) == 0 {
		return nil, execution.ErrPlatformMetaParseIDs
	}

	parseIDs := make([]string, 0, len(rawParseIDs))
	for _, item := range rawParseIDs {
		id, ok := item.(string)
		if !ok {
			return nil, execution.ErrPlatformMetaParseIDs
		}
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		parseIDs = append(parseIDs, id)
	}
	if len(parseIDs) == 0 {
		return nil, execution.ErrPlatformMetaParseIDs
	}

	return parseIDs, nil
}

func (uc *implUseCase) buildJobPayload(specs []execution.DispatchSpec, input execution.DispatchTargetInput) json.RawMessage {
	var queue string
	var action string
	if len(specs) > 0 {
		queue = string(specs[0].Queue)
		action = string(specs[0].Action)
	}

	tasks := make([]map[string]interface{}, 0, len(specs))
	for _, spec := range specs {
		task := map[string]interface{}{
			"queue":  spec.Queue,
			"action": spec.Action,
		}
		if spec.Keyword != "" {
			task["keyword"] = spec.Keyword
		}
		if len(spec.Params) > 0 {
			task["params"] = uc.cloneParams(spec.Params)
		}
		tasks = append(tasks, task)
	}

	payload := map[string]interface{}{
		"queue":      queue,
		"action":     action,
		"task_count": len(specs),
		"tasks":      tasks,
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

func (uc *implUseCase) cloneParams(input map[string]interface{}) map[string]interface{} {
	if len(input) == 0 {
		return nil
	}

	out := make(map[string]interface{}, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}

func (uc *implUseCase) parseInt64(v interface{}) *int64 {
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

func (uc *implUseCase) marshalMetadata(metadata map[string]interface{}) (json.RawMessage, error) {
	if len(metadata) == 0 {
		return nil, nil
	}
	data, err := json.Marshal(metadata)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (uc *implUseCase) isTerminalFailure(status model.JobStatus) bool {
	return status == model.JobStatusFailed || status == model.JobStatusCancelled
}

func (uc *implUseCase) computeEffectiveInterval(source model.DataSource, target model.CrawlTarget) (time.Duration, error) {
	if source.CrawlMode == nil {
		return 0, fmt.Errorf("crawl mode is required")
	}
	if target.CrawlIntervalMinutes <= 0 {
		return 0, fmt.Errorf("crawl interval minutes must be greater than 0")
	}

	multiplier, err := uc.getModeMultiplier(*source.CrawlMode)
	if err != nil {
		return 0, err
	}

	effectiveMinutes := int(math.Round(float64(target.CrawlIntervalMinutes) * multiplier))
	if effectiveMinutes < execution.DefaultMinIntervalMinute {
		effectiveMinutes = execution.DefaultMinIntervalMinute
	}
	if effectiveMinutes > execution.DefaultMaxIntervalMinute {
		effectiveMinutes = execution.DefaultMaxIntervalMinute
	}

	return time.Duration(effectiveMinutes) * time.Minute, nil
}

func (uc *implUseCase) getModeMultiplier(mode model.CrawlMode) (float64, error) {
	switch mode {
	case model.CrawlModeNormal:
		return execution.NormalModeMultiplier, nil
	case model.CrawlModeCrisis:
		return execution.CrisisModeMultiplier, nil
	case model.CrawlModeSleep:
		return execution.SleepModeMultiplier, nil
	default:
		return 0, fmt.Errorf("unsupported crawl mode %s", mode)
	}
}

func (uc *implUseCase) derefCrawlMode(mode *model.CrawlMode) string {
	if mode == nil {
		return ""
	}
	return string(*mode)
}

func (uc *implUseCase) formatTimePtr(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.Format(time.RFC3339)
}

func (uc *implUseCase) extractKeywords(values []string) []string {
	keywords := make([]string, 0, len(values))
	for _, value := range values {
		keyword := strings.TrimSpace(value)
		if keyword == "" {
			continue
		}
		keywords = append(keywords, keyword)
	}
	return keywords
}
