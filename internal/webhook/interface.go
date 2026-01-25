package webhook

import (
	"context"

	"smap-collector/internal/models"
)

//go:generate mockery --name=UseCase
type UseCase interface {
	// NotifyProgress gửi progress update tới Project Service.
	NotifyProgress(ctx context.Context, req ProgressRequest) error

	// NotifyCompletion gửi notification khi project hoàn thành.
	NotifyCompletion(ctx context.Context, req ProgressRequest) error
}

// ProgressNotifier là interface wrapper cho webhook usecase với state integration.
// Kết hợp state usecase và webhook usecase để tự động notify progress.
type ProgressNotifier interface {
	// OnTotalSet được gọi khi xác định được tổng số items.
	// Cập nhật state và notify ngay lập tức.
	OnTotalSet(ctx context.Context, projectID string, total int64) error

	// OnItemDone được gọi sau mỗi item crawl thành công.
	// Cập nhật state và notify.
	OnItemDone(ctx context.Context, projectID string) error

	// OnItemError được gọi sau mỗi item crawl thất bại.
	// Cập nhật state và notify.
	OnItemError(ctx context.Context, projectID string) error

	// OnComplete được gọi khi project hoàn thành (DONE hoặc FAILED).
	// Cập nhật state và notify ngay lập tức.
	OnComplete(ctx context.Context, projectID string, status models.ProjectStatus) error

	// CheckCompletion kiểm tra và tự động complete nếu done + errors >= total.
	// Trả về true nếu project đã complete.
	CheckCompletion(ctx context.Context, projectID string) (bool, error)
}
