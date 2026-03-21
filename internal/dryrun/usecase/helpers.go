package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"ingest-srv/internal/datasource"
	"ingest-srv/internal/dryrun"
	dryrunRepo "ingest-srv/internal/dryrun/repository"
	"ingest-srv/internal/model"

	"github.com/smap-hcmut/shared-libs/go/minio"
)

func (uc *implUseCase) validateTriggerInput(input dryrun.TriggerInput) error {
	if strings.TrimSpace(input.SourceID) == "" {
		return dryrun.ErrSourceNotFound
	}
	if input.SampleLimit != nil && *input.SampleLimit <= 0 {
		return dryrun.ErrInvalidSampleLimit
	}
	return nil
}

func (uc *implUseCase) validateGetLatestInput(input dryrun.GetLatestInput) error {
	if strings.TrimSpace(input.SourceID) == "" {
		return dryrun.ErrSourceNotFound
	}
	return nil
}

func (uc *implUseCase) validateListHistoryInput(input dryrun.ListHistoryInput) error {
	if strings.TrimSpace(input.SourceID) == "" {
		return dryrun.ErrSourceNotFound
	}
	return nil
}

func (uc *implUseCase) validateCompletionInput(input dryrun.HandleCompletionInput) error {
	if strings.TrimSpace(input.TaskID) == "" {
		return dryrun.ErrInvalidCompletionInput
	}

	switch strings.ToLower(strings.TrimSpace(input.Status)) {
	case "success":
		if strings.TrimSpace(input.StorageBucket) == "" || strings.TrimSpace(input.StoragePath) == "" {
			return dryrun.ErrInvalidCompletionInput
		}
	case "error":
	default:
		return dryrun.ErrInvalidCompletionInput
	}

	return nil
}

func (uc *implUseCase) normalizeSampleLimit(limit *int) int {
	if limit == nil || *limit <= 0 {
		return model.DryrunSampleLimitDefault
	}
	return *limit
}

func (uc *implUseCase) parseCompletedAt(raw string) *time.Time {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil
	}

	parsed, err := time.Parse(time.RFC3339, trimmed)
	if err != nil {
		return nil
	}

	result := parsed.UTC()
	return &result
}

func (uc *implUseCase) isTerminalDryrunStatus(status model.DryrunStatus) bool {
	return model.IsTerminalDryrunStatus(status)
}

func (uc *implUseCase) getSource(ctx context.Context, id string) (model.DataSource, error) {
	output, err := uc.dsUC.Detail(ctx, strings.TrimSpace(id))
	if err != nil {
		uc.l.Errorf(ctx, "dryrun.usecase.getSource.dsUC.Detail: id=%s err=%v", id, err)
		return model.DataSource{}, dryrun.ErrSourceNotFound
	}
	if output.DataSource.ID == "" {
		uc.l.Errorf(ctx, "dryrun.usecase.getSource.dsUC.Detail: id=%s err=%v", id, err)
		return model.DataSource{}, dryrun.ErrSourceNotFound
	}
	return output.DataSource, nil
}

func (uc *implUseCase) getTarget(ctx context.Context, dataSourceID, targetID string) (model.CrawlTarget, error) {
	output, err := uc.dsUC.DetailTarget(ctx, datasource.DetailTargetInput{
		DataSourceID: strings.TrimSpace(dataSourceID),
		ID:           strings.TrimSpace(targetID),
	})
	if err != nil {
		uc.l.Errorf(ctx, "dryrun.usecase.getTarget.dsUC.DetailTarget: source_id=%s target_id=%s err=%v", dataSourceID, targetID, err)
		return model.CrawlTarget{}, dryrun.ErrTargetNotFound
	}
	if output.Target.ID == "" {
		return model.CrawlTarget{}, dryrun.ErrTargetNotFound
	}
	return output.Target, nil
}

func (uc *implUseCase) buildKeywordSampleParams(keyword string, sampleLimit int) map[string]interface{} {
	return map[string]interface{}{
		"keyword":                  keyword,
		"limit":                    sampleLimit,
		"threshold":                0.5,
		"comment_count":            50,
		dryrun.ParamKeyRuntimeKind: string(dryrun.RuntimeKindDryrun),
	}
}

func (uc *implUseCase) buildPostDetailParams(parseIDs []string, sampleLimit int) map[string]interface{} {
	return map[string]interface{}{
		"parse_ids":                parseIDs,
		"limit":                    sampleLimit,
		dryrun.ParamKeyRuntimeKind: string(dryrun.RuntimeKindDryrun),
	}
}

func (uc *implUseCase) buildDispatchSpec(source model.DataSource, target *model.CrawlTarget, sampleLimit int) (dryrun.DispatchSpec, json.RawMessage, error) {
	switch {
	case source.SourceType == model.SourceTypeTikTok && source.SourceCategory == model.SourceCategoryCrawl:
		if target == nil || target.TargetType != model.TargetTypeKeyword {
			return dryrun.DispatchSpec{}, nil, dryrun.ErrUnsupportedMapping
		}
		keywords := uc.extractKeywords(target.Values)
		if len(keywords) == 0 {
			return dryrun.DispatchSpec{}, nil, dryrun.ErrUnsupportedMapping
		}

		warnings := make([]map[string]string, 0)
		if len(keywords) > 1 {
			warnings = append(warnings, map[string]string{
				"code":    string(dryrun.WarningCodeMultiValueKeyword),
				"message": "dryrun uses the first keyword value of the target group",
			})
		}

		spec := dryrun.DispatchSpec{
			Queue:  dryrun.QueueNameTikTokTasks,
			Action: dryrun.ActionNameFullFlow,
			Params: uc.buildKeywordSampleParams(keywords[0], sampleLimit),
		}
		if len(warnings) > 0 {
			spec.Params[dryrun.ParamKeyDryrunWarningCode] = warnings[0]["code"]
			spec.Params[dryrun.ParamKeyDryrunWarningMessage] = warnings[0]["message"]
		}
		return spec, uc.marshalWarnings(warnings), nil
	case source.SourceType == model.SourceTypeFacebook && source.SourceCategory == model.SourceCategoryCrawl:
		if target == nil || target.TargetType != model.TargetTypePostURL {
			return dryrun.DispatchSpec{}, nil, dryrun.ErrUnsupportedMapping
		}
		parseIDs, err := uc.parseFacebookParseIDs(target.PlatformMeta)
		if err != nil || len(parseIDs) == 0 {
			return dryrun.DispatchSpec{}, nil, dryrun.ErrUnsupportedMapping
		}
		spec := dryrun.DispatchSpec{
			Queue:  dryrun.QueueNameFacebookTasks,
			Action: dryrun.ActionNamePostDetail,
			Params: uc.buildPostDetailParams(parseIDs, sampleLimit),
		}
		return spec, nil, nil
	default:
		return dryrun.DispatchSpec{}, nil, dryrun.ErrUnsupportedMapping
	}
}

func (uc *implUseCase) publishDispatch(ctx context.Context, input dryrun.PublishDispatchInput) error {
	if uc.publisher == nil {
		return dryrun.ErrDispatchFailed
	}
	if err := uc.publisher.PublishDispatch(ctx, input); err != nil {
		uc.l.Errorf(ctx, "dryrun.usecase.publishDispatch.publisher.PublishDispatch: task_id=%s err=%v", input.TaskID, err)
		return dryrun.ErrDispatchFailed
	}
	return nil
}

func (uc *implUseCase) markDatasourceRunning(ctx context.Context, sourceID, resultID string) (model.DataSource, error) {
	output, err := uc.dsUC.MarkDryrunRunning(ctx, datasource.MarkDryrunRunningInput{
		ID:                 strings.TrimSpace(sourceID),
		DryrunLastResultID: strings.TrimSpace(resultID),
	})
	if err != nil {
		uc.l.Errorf(ctx, "dryrun.usecase.markDatasourceRunning.dsUC.MarkDryrunRunning: source_id=%s result_id=%s err=%v", sourceID, resultID, err)
		return model.DataSource{}, dryrun.ErrUpdateFailed
	}
	return output.DataSource, nil
}

func (uc *implUseCase) applyDatasourceResult(ctx context.Context, sourceID, resultID string, status model.DryrunStatus) (model.DataSource, error) {
	output, err := uc.dsUC.ApplyDryrunResult(ctx, datasource.ApplyDryrunResultInput{
		ID:                 strings.TrimSpace(sourceID),
		DryrunLastResultID: strings.TrimSpace(resultID),
		DryrunStatus:       string(status),
	})
	if err != nil {
		uc.l.Errorf(ctx, "dryrun.usecase.applyDatasourceResult.dsUC.ApplyDryrunResult: source_id=%s result_id=%s status=%s err=%v", sourceID, resultID, status, err)
		return model.DataSource{}, dryrun.ErrUpdateFailed
	}
	return output.DataSource, nil
}

func (uc *implUseCase) failDispatch(ctx context.Context, running model.DryrunResult, errorMessage string) (model.DryrunResult, model.DataSource, error) {
	failedResult, err := uc.repo.UpdateResult(ctx, dryrunRepo.UpdateResultOptions{
		ID:           running.ID,
		Status:       string(model.DryrunStatusFailed),
		SampleCount:  0,
		CompletedAt:  uc.nowPtr(),
		ErrorMessage: strings.TrimSpace(errorMessage),
	})
	if err != nil {
		uc.l.Errorf(ctx, "dryrun.usecase.failDispatch.repo.UpdateResult: result_id=%s err=%v", running.ID, err)
		return model.DryrunResult{}, model.DataSource{}, dryrun.ErrUpdateFailed
	}

	updatedSource, err := uc.applyDatasourceResult(ctx, running.SourceID, failedResult.ID, model.DryrunStatusFailed)
	if err != nil {
		return model.DryrunResult{}, model.DataSource{}, err
	}

	return failedResult, updatedSource, nil
}

func (uc *implUseCase) downloadArtifactBytes(ctx context.Context, bucket, path string) ([]byte, error) {
	if strings.EqualFold(strings.TrimSpace(bucket), "local") {
		data, err := os.ReadFile(strings.TrimSpace(path))
		if err != nil {
			return nil, err
		}
		return data, nil
	}

	if uc.minio == nil {
		return nil, fmt.Errorf("minio client is not initialized")
	}

	reader, _, err := uc.minio.DownloadFile(ctx, &minio.DownloadRequest{
		BucketName: strings.TrimSpace(bucket),
		ObjectName: strings.TrimSpace(path),
	})
	if err != nil {
		return nil, err
	}

	return uc.readAllAndClose(reader)
}

func (uc *implUseCase) readAllAndClose(reader io.ReadCloser) ([]byte, error) {
	if reader == nil {
		return nil, fmt.Errorf("reader is nil")
	}
	defer func() { _ = reader.Close() }()

	return io.ReadAll(reader)
}

func (uc *implUseCase) buildSuccessUpdate(rawBytes []byte, fallbackItemCount *int) (dryrunRepo.UpdateResultOptions, model.DryrunStatus, error) {
	var artifact map[string]interface{}
	if err := json.Unmarshal(rawBytes, &artifact); err != nil {
		warnings := uc.marshalWarnings([]map[string]string{{
			"code":    string(dryrun.WarningCodeInvalidArtifact),
			"message": "dryrun artifact cannot be parsed as JSON",
		}})
		return dryrunRepo.UpdateResultOptions{
			Status:      string(model.DryrunStatusWarning),
			SampleCount: 0,
			Warnings:    warnings,
		}, model.DryrunStatusWarning, nil
	}

	params, _ := artifact["params"].(map[string]interface{})
	resultRaw := uc.marshalJSON(artifact["result"])
	artifactItemCount := uc.parseInt(artifact["item_count"])

	sampleLimit := uc.normalizeSampleLimit(nil)
	if rawLimit, ok := params["limit"]; ok {
		if parsed := uc.parseInt(rawLimit); parsed > 0 {
			sampleLimit = parsed
		}
	}

	sampleData, sampleCount, totalFound, warnings := uc.buildSamplePayload(resultRaw, sampleLimit, artifactItemCount, fallbackItemCount)
	warnings = uc.appendWarningFromParams(params, warnings)
	finalStatus := model.DryrunStatusSuccess
	if len(warnings) > 0 {
		finalStatus = model.DryrunStatusWarning
	}
	return dryrunRepo.UpdateResultOptions{
		Status:      string(finalStatus),
		SampleCount: sampleCount,
		TotalFound:  totalFound,
		SampleData:  sampleData,
		Warnings:    warnings,
	}, finalStatus, nil
}

func (uc *implUseCase) buildSamplePayload(resultRaw json.RawMessage, sampleLimit int, artifactItemCount int, fallbackItemCount *int) (json.RawMessage, int, *int, json.RawMessage) {
	if len(resultRaw) == 0 || string(resultRaw) == "null" {
		return nil, 0, uc.normalizeTotalFound(artifactItemCount, fallbackItemCount), uc.marshalWarnings([]map[string]string{{
			"code":    string(dryrun.WarningCodeNoSampleData),
			"message": "crawler returned success without sample result payload",
		}})
	}

	var payload interface{}
	if err := json.Unmarshal(resultRaw, &payload); err != nil {
		return nil, 0, uc.normalizeTotalFound(artifactItemCount, fallbackItemCount), uc.marshalWarnings([]map[string]string{{
			"code":    string(dryrun.WarningCodeInvalidArtifact),
			"message": "crawler result payload is not valid JSON",
		}})
	}

	switch typed := payload.(type) {
	case []interface{}:
		sample := uc.truncateItems(typed, sampleLimit)
		return uc.marshalJSON(sample), len(sample), uc.intPtr(len(typed)), nil
	case map[string]interface{}:
		totalFound := uc.normalizeTotalFound(artifactItemCount, fallbackItemCount)
		for _, key := range []string{"posts", "items", "data", "results", "comments", "videos"} {
			if items, ok := typed[key].([]interface{}); ok {
				sample := uc.truncateItems(items, sampleLimit)
				if totalFound == nil {
					totalFound = uc.intPtr(len(items))
				}
				if rawTotal := uc.parseInt(typed["total_posts"]); rawTotal > 0 {
					totalFound = uc.intPtr(rawTotal)
				}
				if rawTotal := uc.parseInt(typed["item_count"]); rawTotal > 0 {
					totalFound = uc.intPtr(rawTotal)
				}
				if len(sample) == 0 {
					return nil, 0, totalFound, uc.marshalWarnings([]map[string]string{{
						"code":    string(dryrun.WarningCodeNoSampleData),
						"message": "crawler returned an empty collection",
					}})
				}
				return uc.marshalJSON(sample), len(sample), totalFound, nil
			}
		}

		return uc.marshalJSON([]interface{}{typed}), 1, uc.normalizeTotalFoundFromObject(typed, totalFound), uc.marshalWarnings([]map[string]string{{
			"code":    string(dryrun.WarningCodeObjectSampleFallback),
			"message": "crawler result payload is an object; dryrun stores the object as a single sample",
		}})
	default:
		return uc.marshalJSON([]interface{}{typed}), 1, uc.normalizeTotalFound(artifactItemCount, fallbackItemCount), uc.marshalWarnings([]map[string]string{{
			"code":    string(dryrun.WarningCodeObjectSampleFallback),
			"message": "crawler result payload is scalar; dryrun stores it as a single sample",
		}})
	}
}

func (uc *implUseCase) truncateItems(items []interface{}, sampleLimit int) []interface{} {
	if len(items) == 0 {
		return nil
	}
	if sampleLimit <= 0 || sampleLimit >= len(items) {
		return items
	}
	return items[:sampleLimit]
}

func (uc *implUseCase) marshalJSON(input interface{}) json.RawMessage {
	if input == nil {
		return nil
	}
	data, err := json.Marshal(input)
	if err != nil {
		return nil
	}
	return data
}

func (uc *implUseCase) marshalWarnings(warnings []map[string]string) json.RawMessage {
	if len(warnings) == 0 {
		return nil
	}
	return uc.marshalJSON(warnings)
}

func (uc *implUseCase) appendWarningFromParams(params map[string]interface{}, warnings json.RawMessage) json.RawMessage {
	code, _ := params[dryrun.ParamKeyDryrunWarningCode].(string)
	message, _ := params[dryrun.ParamKeyDryrunWarningMessage].(string)
	if strings.TrimSpace(code) == "" || strings.TrimSpace(message) == "" {
		return warnings
	}

	items := make([]map[string]string, 0)
	if len(warnings) > 0 {
		_ = json.Unmarshal(warnings, &items)
	}
	items = append(items, map[string]string{
		"code":    strings.TrimSpace(code),
		"message": strings.TrimSpace(message),
	})
	return uc.marshalWarnings(items)
}

func (uc *implUseCase) ensureActivatedTargetAfterUsableDryrun(ctx context.Context, result model.DryrunResult) error {
	if !model.IsUsableDryrunStatus(result.Status) {
		return nil
	}
	if strings.TrimSpace(result.TargetID) == "" {
		return nil
	}

	_, err := uc.dsUC.ActivateTarget(ctx, datasource.ActivateTargetInput{
		DataSourceID: strings.TrimSpace(result.SourceID),
		ID:           strings.TrimSpace(result.TargetID),
	})
	if err != nil {
		if errors.Is(err, datasource.ErrTargetNotFound) || errors.Is(err, datasource.ErrTargetActivateNotAllowed) {
			uc.l.Warnf(ctx, "dryrun.usecase.ensureActivatedTargetAfterUsableDryrun.skip_activate: source_id=%s target_id=%s err=%v", result.SourceID, result.TargetID, err)
			return nil
		}
		uc.l.Errorf(ctx, "dryrun.usecase.ensureActivatedTargetAfterUsableDryrun.dsUC.ActivateTarget: source_id=%s target_id=%s err=%v", result.SourceID, result.TargetID, err)
		return dryrun.ErrUpdateFailed
	}

	return nil
}

func (uc *implUseCase) normalizeTotalFound(artifactItemCount int, fallbackItemCount *int) *int {
	if artifactItemCount > 0 {
		return uc.intPtr(artifactItemCount)
	}
	if fallbackItemCount != nil && *fallbackItemCount > 0 {
		return uc.intPtr(*fallbackItemCount)
	}
	return nil
}

func (uc *implUseCase) normalizeTotalFoundFromObject(payload map[string]interface{}, fallback *int) *int {
	if total := uc.parseInt(payload["total_posts"]); total > 0 {
		return uc.intPtr(total)
	}
	if total := uc.parseInt(payload["item_count"]); total > 0 {
		return uc.intPtr(total)
	}
	return fallback
}

func (uc *implUseCase) parseInt(value interface{}) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int32:
		return int(typed)
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	case json.Number:
		v, _ := typed.Int64()
		return int(v)
	}
	return 0
}

func (uc *implUseCase) intPtr(v int) *int {
	value := v
	return &value
}

func (uc *implUseCase) extractKeywords(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		result = append(result, trimmed)
	}
	return result
}

func (uc *implUseCase) parseFacebookParseIDs(platformMeta json.RawMessage) ([]string, error) {
	if len(platformMeta) == 0 {
		return nil, dryrun.ErrUnsupportedMapping
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(platformMeta, &payload); err != nil {
		return nil, dryrun.ErrUnsupportedMapping
	}

	rawParseIDs, ok := payload["parse_ids"].([]interface{})
	if !ok || len(rawParseIDs) == 0 {
		return nil, dryrun.ErrUnsupportedMapping
	}

	parseIDs := make([]string, 0, len(rawParseIDs))
	for _, item := range rawParseIDs {
		id, ok := item.(string)
		if !ok {
			return nil, dryrun.ErrUnsupportedMapping
		}
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		parseIDs = append(parseIDs, id)
	}
	if len(parseIDs) == 0 {
		return nil, dryrun.ErrUnsupportedMapping
	}

	return parseIDs, nil
}

func (uc *implUseCase) nowPtr() *time.Time {
	now := uc.now()
	return &now
}
