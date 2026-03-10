package repository

import "context"

type Repository interface {
	ClaimRawBatchForParsing(ctx context.Context, rawBatchID string) (bool, error)
	MarkRawBatchDownloaded(ctx context.Context, opt MarkRawBatchDownloadedOptions) error
	MarkRawBatchParsed(ctx context.Context, opt MarkRawBatchParsedOptions) error
	MarkRawBatchFailed(ctx context.Context, opt MarkRawBatchFailedOptions) error
}
