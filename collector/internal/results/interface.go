package results

import (
	"context"

	"smap-collector/internal/models"
)

//go:generate mockery --name=ResultUseCase
type ResultUseCase interface {
	// HandleResult xử lý kết quả từ worker, quyết định retry/ack và cập nhật trạng thái.
	HandleResult(ctx context.Context, res models.CrawlerResult) error
}

// UseCase export cho module results (fan-in + retry).
// Note: DataCollected event publishing is handled by Crawler services, not Collector.
type UseCase interface {
	ResultUseCase
}
