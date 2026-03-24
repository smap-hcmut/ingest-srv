package usecase

import (
	"context"

	"ingest-srv/internal/uap"
)

func (uc *implUseCase) publishRecord(
	ctx context.Context,
	record uap.UAPRecord,
	input uap.ParseAndStoreRawBatchInput,
	stats *uap.KafkaPublishStats,
) {
	if stats == nil {
		return
	}

	if uc.publisher == nil {
		return
	}

	// stats.AttemptedCount++
	// if err := uc.publisher.Publish(ctx, uap.PublishUAPInput{
	// 	Record: record,
	// }); err != nil {
	// 	stats.FailedCount++
	// 	stats.LastError = err.Error()
	// 	if uc.l != nil {
	// 		uc.l.Warnf(ctx, "uap.usecase.publishRecord.Publish: raw_batch_id=%s uap_id=%s err=%v", input.RawBatchID, record.Identity.UAPID, err)
	// 	}
	// 	return
	// }

	// stats.SuccessCount++
	// if uc.l != nil {
	// 	uc.l.Infof(ctx, "uap.usecase.publishRecord.success: raw_batch_id=%s uap_id=%s topic=%s", input.RawBatchID, record.Identity.UAPID, stats.Topic)
	// }
}
