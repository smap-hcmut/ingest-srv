package usecase

import (
	"context"
	"fmt"
	"ingest-srv/internal/uap"
	repo "ingest-srv/internal/uap/repository"
	"strings"

	"github.com/smap-hcmut/shared-libs/go/minio"
)

func (uc *implUseCase) ParseAndStoreRawBatch(ctx context.Context, input uap.ParseAndStoreRawBatchInput) error {
	if err := validateParseAndStoreRawBatchInput(input); err != nil {
		return err
	}

	claimed, err := uc.repo.ClaimRawBatchForParsing(ctx, input.RawBatchID)
	if err != nil {
		uc.l.Errorf(ctx, "uap.usecase.ParseAndStoreRawBatch.ClaimRawBatchForParsing: raw_batch_id=%s err=%v", input.RawBatchID, err)
		return err
	}
	if !claimed {
		uc.l.Infof(ctx, "uap.usecase.ParseAndStoreRawBatch: raw_batch_id=%s already claimed or already processed", input.RawBatchID)
		return nil
	}

	reader, _, err := uc.minio.DownloadFile(ctx, &minio.DownloadRequest{
		BucketName: input.StorageBucket,
		ObjectName: input.StoragePath,
	})
	if err != nil {
		errMessage := fmt.Sprintf("download raw batch object: %v", err)
		_ = failRawBatch(ctx, uc, input, errMessage, "", nil, 0, nil)
		return err
	}

	rawBytes, err := readAllAndClose(reader)
	if err != nil {
		errMessage := fmt.Sprintf("read raw batch object: %v", err)
		_ = failRawBatch(ctx, uc, input, errMessage, "", nil, 0, nil)
		return err
	}

	if err := uc.repo.MarkRawBatchDownloaded(ctx, repo.MarkRawBatchDownloadedOptions{
		RawBatchID: input.RawBatchID,
	}); err != nil {
		_ = failRawBatch(ctx, uc, input, fmt.Sprintf("mark raw batch downloaded: %v", err), "", nil, 0, nil)
		return err
	}

	publishStats := &kafkaPublishStats{
		Topic: strings.TrimSpace(uc.uapTopic),
	}
	if uc.publisher == nil {
		uc.l.Warnf(ctx, "uap.usecase.ParseAndStoreRawBatch: kafka publisher is disabled for raw_batch_id=%s", input.RawBatchID)
	}

	records, err := flattenTikTokFullFlow(rawBytes, input, func(record uap.UAPRecord) {
		publishRecord(ctx, uc, record, input, publishStats)
	})
	if err != nil {
		errMessage := fmt.Sprintf("parse tiktok full_flow raw batch: %v", err)
		_ = failRawBatch(ctx, uc, input, errMessage, "", nil, 0, publishStats)
		return err
	}

	outputBucket := strings.TrimSpace(uc.outputBucket)
	if outputBucket == "" {
		outputBucket = input.StorageBucket
	}

	chunks := chunkRecords(records)
	parts := make([]artifactPart, 0, len(chunks))
	for index, chunk := range chunks {
		part, uploadErr := uploadChunk(ctx, uc.minio, outputBucket, input.ProjectID, input.SourceID, input.BatchID, index+1, chunk)
		if uploadErr != nil {
			publishErr := fmt.Sprintf("upload uap chunk: %v", uploadErr)
			_ = failRawBatch(ctx, uc, input, "", publishErr, parts, len(records), publishStats)
			return uploadErr
		}
		parts = append(parts, part)
	}

	metadata, err := mergeRawMetadata(input.RawMetadata, parts, len(records), publishStats)
	if err != nil {
		_ = failRawBatch(ctx, uc, input, fmt.Sprintf("merge parsed raw metadata: %v", err), "", parts, len(records), publishStats)
		return err
	}

	if err := uc.repo.MarkRawBatchParsed(ctx, repo.MarkRawBatchParsedOptions{
		RawBatchID:         input.RawBatchID,
		ParsedAt:           uc.now(),
		PublishRecordCount: len(records),
		RawMetadata:        metadata,
	}); err != nil {
		_ = failRawBatch(ctx, uc, input, fmt.Sprintf("mark raw batch parsed: %v", err), "", parts, len(records), publishStats)
		return err
	}

	uc.l.Infof(
		ctx,
		"uap.usecase.ParseAndStoreRawBatch.success: raw_batch_id=%s total_records=%d total_parts=%d kafka_publish_attempted=%d kafka_publish_success=%d kafka_publish_failed=%d",
		input.RawBatchID,
		len(records),
		len(parts),
		publishStats.AttemptedCount,
		publishStats.SuccessCount,
		publishStats.FailedCount,
	)
	return nil
}
