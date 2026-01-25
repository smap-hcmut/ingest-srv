package usecase

import (
	"context"

	"smap-collector/internal/models"
)

// Producer triển khai interface dispatcher.Producer ngay trong implUseCase.
func (uc implUseCase) PublishTikTokTask(ctx context.Context, task models.TikTokCollectorTask) error {
	return uc.prod.PublishTikTokTask(ctx, task)
}

func (uc implUseCase) PublishYouTubeTask(ctx context.Context, task models.YouTubeCollectorTask) error {
	return uc.prod.PublishYouTubeTask(ctx, task)
}
