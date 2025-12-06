package state

import (
	"context"

	"smap-collector/internal/models"
)

//go:generate mockery --name=UseCase
type UseCase interface {
	// InitState khởi tạo state cho project mới với status INITIALIZING.
	// Được gọi khi nhận ProjectCreatedEvent.
	InitState(ctx context.Context, projectID string) error

	// UpdateTotal cập nhật tổng số items cần crawl và chuyển status sang CRAWLING.
	// Được gọi khi Collector xác định được tổng số tasks.
	UpdateTotal(ctx context.Context, projectID string, total int64) error

	// IncrementDone tăng counter done lên 1.
	// Được gọi sau mỗi item crawl thành công.
	IncrementDone(ctx context.Context, projectID string) error

	// IncrementErrors tăng counter errors lên 1.
	// Được gọi sau mỗi item crawl thất bại.
	IncrementErrors(ctx context.Context, projectID string) error

	// UpdateStatus cập nhật status của project.
	// Dùng để set DONE, FAILED, hoặc các status khác.
	UpdateStatus(ctx context.Context, projectID string, status models.ProjectStatus) error

	// GetState lấy state hiện tại của project.
	// Trả về nil nếu project không tồn tại.
	GetState(ctx context.Context, projectID string) (*models.ProjectState, error)

	// CheckAndUpdateCompletion kiểm tra nếu done + errors >= total thì update status DONE.
	// Trả về true nếu project đã complete.
	CheckAndUpdateCompletion(ctx context.Context, projectID string) (bool, error)

	// StoreUserMapping lưu mapping project_id -> user_id.
	// Dùng để lookup user_id khi cần notify progress.
	StoreUserMapping(ctx context.Context, projectID, userID string) error

	// GetUserID lấy user_id từ project_id.
	GetUserID(ctx context.Context, projectID string) (string, error)
}
