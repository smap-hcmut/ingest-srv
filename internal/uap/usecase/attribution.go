package usecase

import (
	"encoding/json"
	"strings"

	"ingest-srv/internal/uap"
)

func (uc *implUseCase) withIngestAttribution(record uap.UAPRecord, input uap.ParseAndStoreRawBatchInput) uap.UAPRecord {
	if record.PlatformMeta == nil {
		record.PlatformMeta = make(map[string]interface{})
	}

	params := uc.requestParams(input.RequestPayload)
	sourceKind := uc.resolveSourceKind(input, params)
	smap := map[string]interface{}{
		"data_source_id":   strings.TrimSpace(input.SourceID),
		"target_id":        strings.TrimSpace(input.TargetID),
		"external_task_id": strings.TrimSpace(input.ExternalTaskID),
		"task_id":          strings.TrimSpace(input.TaskID),
		"source_kind":      sourceKind,
	}

	for _, key := range []string{"page_id", "profile_url", "username", "sec_uid", "keyword"} {
		if value := strings.TrimSpace(uc.stringParam(params, key)); value != "" {
			smap[key] = value
		}
	}

	record.PlatformMeta["smap"] = smap
	return record
}

func (uc *implUseCase) requestParams(raw json.RawMessage) map[string]interface{} {
	if len(raw) == 0 {
		return nil
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil
	}
	params, _ := payload["params"].(map[string]interface{})
	return params
}

func (uc *implUseCase) resolveSourceKind(input uap.ParseAndStoreRawBatchInput, params map[string]interface{}) string {
	if value := strings.TrimSpace(uc.stringParam(params, "source_kind")); value != "" {
		return value
	}
	switch strings.TrimSpace(input.Action) {
	case uap.TaskTypePageFullFlow:
		return "focused_page"
	case uap.TaskTypeUserFullFlow:
		return "focused_profile"
	case uap.TaskTypeFullFlow:
		return "keyword_search"
	default:
		return strings.TrimSpace(input.Action)
	}
}

func (uc *implUseCase) stringParam(params map[string]interface{}, key string) string {
	if params == nil {
		return ""
	}
	switch value := params[key].(type) {
	case string:
		return value
	default:
		return ""
	}
}
