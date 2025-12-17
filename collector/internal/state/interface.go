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

	// ============================================================================
	// Task-Level Methods (for completion check)
	// ============================================================================

	// SetTasksTotal set tổng số tasks và items expected, chuyển status sang PROCESSING.
	// tasksTotal = số tasks dispatch (keywords × platforms)
	// itemsExpected = tasksTotal × limitPerKeyword
	SetTasksTotal(ctx context.Context, projectID string, tasksTotal, itemsExpected int64) error

	// IncrementTasksDone tăng counter tasks_done lên 1.
	// Được gọi khi nhận response thành công từ Crawler.
	IncrementTasksDone(ctx context.Context, projectID string) error

	// IncrementTasksErrors tăng counter tasks_errors lên 1.
	// Được gọi khi nhận response thất bại từ Crawler.
	IncrementTasksErrors(ctx context.Context, projectID string) error

	// ============================================================================
	// Item-Level Methods (for progress display)
	// ============================================================================

	// IncrementItemsActualBy tăng counter items_actual lên N.
	// Được gọi với stats.successful từ Crawler response.
	IncrementItemsActualBy(ctx context.Context, projectID string, count int64) error

	// IncrementItemsErrorsBy tăng counter items_errors lên N.
	// Được gọi với stats.failed từ Crawler response.
	IncrementItemsErrorsBy(ctx context.Context, projectID string, count int64) error

	// ============================================================================
	// Legacy Crawl Phase Methods (for backward compatibility)
	// Deprecated: Use task-level and item-level methods instead
	// ============================================================================

	// SetCrawlTotal set tổng số items cần crawl và chuyển status sang PROCESSING.
	// Deprecated: Use SetTasksTotal instead.
	SetCrawlTotal(ctx context.Context, projectID string, total int64) error

	// IncrementCrawlDoneBy tăng counter crawl_done lên N.
	// Deprecated: Use IncrementTasksDone and IncrementItemsActualBy instead.
	IncrementCrawlDoneBy(ctx context.Context, projectID string, count int64) error

	// IncrementCrawlErrorsBy tăng counter crawl_errors lên N.
	// Deprecated: Use IncrementTasksErrors and IncrementItemsErrorsBy instead.
	IncrementCrawlErrorsBy(ctx context.Context, projectID string, count int64) error

	// ============================================================================
	// Analyze Phase Methods
	// ============================================================================

	// IncrementAnalyzeTotalBy tăng counter analyze_total lên N.
	// Được gọi khi crawl thành công (mỗi item crawl thành công = 1 item cần analyze).
	IncrementAnalyzeTotalBy(ctx context.Context, projectID string, count int64) error

	// IncrementAnalyzeDoneBy tăng counter analyze_done lên N.
	// Được gọi sau mỗi batch analyze thành công.
	IncrementAnalyzeDoneBy(ctx context.Context, projectID string, count int64) error

	// IncrementAnalyzeErrorsBy tăng counter analyze_errors lên N.
	// Được gọi sau mỗi batch analyze thất bại.
	IncrementAnalyzeErrorsBy(ctx context.Context, projectID string, count int64) error

	// ============================================================================
	// Status & State Methods
	// ============================================================================

	// UpdateStatus cập nhật status của project.
	// Dùng để set DONE, FAILED, hoặc các status khác.
	UpdateStatus(ctx context.Context, projectID string, status models.ProjectStatus) error

	// GetState lấy state hiện tại của project.
	// Trả về nil nếu project không tồn tại.
	GetState(ctx context.Context, projectID string) (*models.ProjectState, error)

	// CheckCompletion kiểm tra nếu cả crawl và analyze đều complete thì update status DONE.
	// Trả về true nếu project đã complete.
	CheckCompletion(ctx context.Context, projectID string) (bool, error)

	// ============================================================================
	// User Mapping Methods
	// ============================================================================

	// StoreUserMapping lưu mapping project_id -> user_id.
	// Dùng để lookup user_id khi cần notify progress.
	StoreUserMapping(ctx context.Context, projectID, userID string) error

	// GetUserID lấy user_id từ project_id.
	GetUserID(ctx context.Context, projectID string) (string, error)
}
