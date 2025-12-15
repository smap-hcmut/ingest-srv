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
	// Crawl Phase Methods
	// ============================================================================

	// SetCrawlTotal set tổng số items cần crawl và chuyển status sang PROCESSING.
	// Được gọi khi Collector xác định được tổng số tasks.
	SetCrawlTotal(ctx context.Context, projectID string, total int64) error

	// IncrementCrawlDoneBy tăng counter crawl_done lên N.
	// Được gọi sau mỗi batch crawl thành công.
	IncrementCrawlDoneBy(ctx context.Context, projectID string, count int64) error

	// IncrementCrawlErrorsBy tăng counter crawl_errors lên N.
	// Được gọi sau mỗi batch crawl thất bại.
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
