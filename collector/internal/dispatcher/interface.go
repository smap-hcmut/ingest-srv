package dispatcher

import (
	"context"

	"smap-collector/internal/models"
)

//go:generate mockery --name=UseCase
type UseCase interface {
	Dispatch(ctx context.Context, req models.CrawlRequest) ([]models.CollectorTask, error)
	HandleProjectCreatedEvent(ctx context.Context, event models.ProjectCreatedEvent) error
	Producer
}

// Producer is used by the use case layer to publish tasks (implemented in the delivery layer).
type Producer interface {
	PublishTikTokTask(ctx context.Context, task models.TikTokCollectorTask) error
	PublishYouTubeTask(ctx context.Context, task models.YouTubeCollectorTask) error
}
