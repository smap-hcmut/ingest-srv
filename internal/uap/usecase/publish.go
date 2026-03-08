package usecase

import (
	"context"
	"encoding/json"
	"strings"

	"ingest-srv/internal/uap"
)

type kafkaPublishStats struct {
	Topic          string
	AttemptedCount int
	SuccessCount   int
	FailedCount    int
	LastError      string
}

func publishRecord(
	ctx context.Context,
	uc *implUseCase,
	record uap.UAPRecord,
	input uap.ParseAndStoreRawBatchInput,
	stats *kafkaPublishStats,
) {
	if stats == nil {
		return
	}

	if uc.publisher == nil {
		return
	}

	body, err := json.Marshal(record)
	stats.AttemptedCount++
	if err != nil {
		stats.FailedCount++
		stats.LastError = err.Error()
		uc.l.Warnf(ctx, "uap.usecase.publishRecord.marshal: raw_batch_id=%s uap_id=%s err=%v", input.RawBatchID, record.Identity.UAPID, err)
		return
	}

	if err := uc.publisher.Publish(ctx, uap.PublishUAPInput{
		Key:   []byte(strings.TrimSpace(record.Identity.UAPID)),
		Value: body,
	}); err != nil {
		stats.FailedCount++
		stats.LastError = err.Error()
		uc.l.Warnf(ctx, "uap.usecase.publishRecord.Publish: raw_batch_id=%s uap_id=%s err=%v", input.RawBatchID, record.Identity.UAPID, err)
		return
	}

	stats.SuccessCount++
	uc.l.Infof(ctx, "uap.usecase.publishRecord.success: raw_batch_id=%s uap_id=%s topic=%s", input.RawBatchID, record.Identity.UAPID, stats.Topic)
}
