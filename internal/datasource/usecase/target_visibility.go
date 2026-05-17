package usecase

import (
	"encoding/json"
	"time"

	"ingest-srv/internal/model"
)

func targetFlushedPlatformMeta(target model.CrawlTarget, now time.Time) json.RawMessage {
	meta := map[string]interface{}{}
	if len(target.PlatformMeta) > 0 {
		_ = json.Unmarshal(target.PlatformMeta, &meta)
	}

	smapMeta := map[string]interface{}{}
	if raw, ok := meta["smap"].(map[string]interface{}); ok {
		for key, value := range raw {
			smapMeta[key] = value
		}
	}
	smapMeta["deleted_at"] = now.UTC().Format(time.RFC3339)
	smapMeta["visibility"] = "flushed"
	meta["smap"] = smapMeta

	encoded, err := json.Marshal(meta)
	if err != nil {
		return target.PlatformMeta
	}
	return encoded
}
