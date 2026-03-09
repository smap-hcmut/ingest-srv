package postgre

import (
	"context"
	"strings"

	repo "ingest-srv/internal/uap/repository"
	"ingest-srv/internal/model"
	"ingest-srv/internal/sqlboiler"

	"github.com/aarondl/null/v8"
	"github.com/aarondl/sqlboiler/v4/boil"
)

func (r *implRepository) ClaimRawBatchForParsing(ctx context.Context, rawBatchID string) (bool, error) {
	rowsAffected, err := sqlboiler.RawBatches(
		sqlboiler.RawBatchWhere.ID.EQ(strings.TrimSpace(rawBatchID)),
		sqlboiler.RawBatchWhere.Status.EQ(sqlboiler.BatchStatus(model.BatchStatusReceived)),
		sqlboiler.RawBatchWhere.PublishStatus.EQ(sqlboiler.PublishStatus(model.PublishStatusPending)),
	).UpdateAll(ctx, r.db, sqlboiler.M{
		sqlboiler.RawBatchColumns.PublishStatus: sqlboiler.PublishStatus(model.PublishStatusPublishing),
		sqlboiler.RawBatchColumns.PublishError:  null.String{},
	})
	if err != nil {
		r.l.Errorf(ctx, "uap.repository.ClaimRawBatchForParsing.UpdateAll: %v", err)
		return false, repo.ErrClaimRawBatch
	}

	return rowsAffected > 0, nil
}

func (r *implRepository) MarkRawBatchDownloaded(ctx context.Context, opt repo.MarkRawBatchDownloadedOptions) error {
	row, err := sqlboiler.FindRawBatch(ctx, r.db, opt.RawBatchID)
	if err != nil {
		r.l.Errorf(ctx, "uap.repository.MarkRawBatchDownloaded.FindRawBatch: %v", err)
		return repo.ErrRawBatchNotFound
	}

	row.Status = sqlboiler.BatchStatus(model.BatchStatusDownloaded)
	if _, err := row.Update(ctx, r.db, boil.Whitelist(
		sqlboiler.RawBatchColumns.Status,
	)); err != nil {
		r.l.Errorf(ctx, "uap.repository.MarkRawBatchDownloaded.Update: %v", err)
		return repo.ErrUpdateRawBatch
	}

	return nil
}

func (r *implRepository) MarkRawBatchParsed(ctx context.Context, opt repo.MarkRawBatchParsedOptions) error {
	row, err := sqlboiler.FindRawBatch(ctx, r.db, opt.RawBatchID)
	if err != nil {
		r.l.Errorf(ctx, "uap.repository.MarkRawBatchParsed.FindRawBatch: %v", err)
		return repo.ErrRawBatchNotFound
	}

	row.Status = sqlboiler.BatchStatus(model.BatchStatusParsed)
	row.ParsedAt = null.TimeFrom(opt.ParsedAt)
	row.PublishStatus = sqlboiler.PublishStatus(model.PublishStatusPending)
	row.PublishRecordCount = opt.PublishRecordCount
	row.ErrorMessage = null.String{}
	row.PublishError = null.String{}
	if len(opt.RawMetadata) > 0 {
		row.RawMetadata = null.JSONFrom(opt.RawMetadata)
	}

	if _, err := row.Update(ctx, r.db, boil.Whitelist(
		sqlboiler.RawBatchColumns.Status,
		sqlboiler.RawBatchColumns.ParsedAt,
		sqlboiler.RawBatchColumns.PublishStatus,
		sqlboiler.RawBatchColumns.PublishRecordCount,
		sqlboiler.RawBatchColumns.ErrorMessage,
		sqlboiler.RawBatchColumns.PublishError,
		sqlboiler.RawBatchColumns.RawMetadata,
	)); err != nil {
		r.l.Errorf(ctx, "uap.repository.MarkRawBatchParsed.Update: %v", err)
		return repo.ErrUpdateRawBatch
	}

	return nil
}

func (r *implRepository) MarkRawBatchFailed(ctx context.Context, opt repo.MarkRawBatchFailedOptions) error {
	row, err := sqlboiler.FindRawBatch(ctx, r.db, opt.RawBatchID)
	if err != nil {
		r.l.Errorf(ctx, "uap.repository.MarkRawBatchFailed.FindRawBatch: %v", err)
		return repo.ErrRawBatchNotFound
	}

	row.Status = sqlboiler.BatchStatus(model.BatchStatusFailed)
	row.PublishStatus = sqlboiler.PublishStatus(model.PublishStatusPending)
	if strings.TrimSpace(opt.ErrorMessage) == "" {
		row.ErrorMessage = null.String{}
	} else {
		row.ErrorMessage = null.StringFrom(strings.TrimSpace(opt.ErrorMessage))
	}
	if strings.TrimSpace(opt.PublishError) == "" {
		row.PublishError = null.String{}
	} else {
		row.PublishError = null.StringFrom(strings.TrimSpace(opt.PublishError))
	}
	if len(opt.RawMetadata) > 0 {
		row.RawMetadata = null.JSONFrom(opt.RawMetadata)
	}

	if _, err := row.Update(ctx, r.db, boil.Whitelist(
		sqlboiler.RawBatchColumns.Status,
		sqlboiler.RawBatchColumns.PublishStatus,
		sqlboiler.RawBatchColumns.ErrorMessage,
		sqlboiler.RawBatchColumns.PublishError,
		sqlboiler.RawBatchColumns.RawMetadata,
	)); err != nil {
		r.l.Errorf(ctx, "uap.repository.MarkRawBatchFailed.Update: %v", err)
		return repo.ErrUpdateRawBatch
	}

	return nil
}
