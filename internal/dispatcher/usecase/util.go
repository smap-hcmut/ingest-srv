package usecase

import (
	"encoding/json"
	"time"

	"smap-collector/internal/dispatcher"
	"smap-collector/internal/models"
)

func (uc implUseCase) selectPlatforms() []models.Platform {
	platforms := make([]models.Platform, 0, len(uc.defaultOptions.PlatformQueues))
	for p := range uc.defaultOptions.PlatformQueues {
		platforms = append(platforms, p)
	}
	return platforms
}

func (uc implUseCase) mapPayload(platform models.Platform, taskType models.TaskType, raw map[string]any) (any, error) {
	if raw == nil {
		return nil, dispatcher.ErrInvalidInput
	}

	switch platform {
	case models.PlatformYouTube:
		return uc.mapYouTubePayload(taskType, raw)
	case models.PlatformTikTok:
		return uc.mapTikTokPayload(taskType, raw)
	default:
		return nil, dispatcher.ErrUnknownRoute
	}
}

func (uc implUseCase) mapYouTubePayload(taskType models.TaskType, raw map[string]any) (any, error) {
	switch taskType {
	case models.TaskTypeResearchKeyword:
		var payload models.YouTubeResearchKeywordPayload
		return uc.decodePayload(raw, &payload)
	case models.TaskTypeCrawlLinks:
		var payload models.YouTubeCrawlLinksPayload
		return uc.decodePayload(raw, &payload)
	case models.TaskTypeResearchAndCrawl:
		var payload models.YouTubeResearchAndCrawlPayload
		return uc.decodePayload(raw, &payload)
	case models.TaskTypeDryRunKeyword:
		var payload models.YouTubeResearchAndCrawlPayload
		// Map dry-run payload with config-driven limits
		_, err := uc.decodePayload(raw, &payload)
		if err != nil {
			return nil, err
		}
		// Enforce dry-run limits from config
		payload.LimitPerKeyword = uc.crawlLimitsCfg.DryRunLimitPerKeyword
		payload.IncludeComments = uc.crawlLimitsCfg.IncludeComments
		payload.MaxComments = uc.crawlLimitsCfg.DryRunMaxComments
		return &payload, nil
	default:
		return nil, dispatcher.ErrUnknownRoute
	}
}

func (uc implUseCase) mapTikTokPayload(taskType models.TaskType, raw map[string]any) (any, error) {
	switch taskType {
	case models.TaskTypeResearchKeyword:
		var payload models.TikTokResearchKeywordPayload
		return uc.decodePayload(raw, &payload)
	case models.TaskTypeCrawlLinks:
		var payload models.TikTokCrawlLinksPayload
		return uc.decodePayload(raw, &payload)
	case models.TaskTypeResearchAndCrawl:
		var payload models.TikTokResearchAndCrawlPayload
		return uc.decodePayload(raw, &payload)
	case models.TaskTypeDryRunKeyword:
		var payload models.TikTokResearchAndCrawlPayload
		// Map dry-run payload with config-driven limits
		_, err := uc.decodePayload(raw, &payload)
		if err != nil {
			return nil, err
		}
		// Enforce dry-run limits from config
		payload.LimitPerKeyword = uc.crawlLimitsCfg.DryRunLimitPerKeyword
		payload.IncludeComments = uc.crawlLimitsCfg.IncludeComments
		payload.MaxComments = uc.crawlLimitsCfg.DryRunMaxComments
		return &payload, nil
	default:
		return nil, dispatcher.ErrUnknownRoute
	}
}

func (uc implUseCase) decodePayload(raw map[string]any, dest any) (any, error) {
	b, err := json.Marshal(raw)
	if err != nil {
		return nil, dispatcher.ErrInvalidInput
	}
	if err := json.Unmarshal(b, dest); err != nil {
		return nil, dispatcher.ErrInvalidInput
	}
	return dest, nil
}

// validateRequest sets defaults for attempt/max/timestamps.
func (uc implUseCase) validateRequest(req *models.CrawlRequest, opts dispatcher.Options) error {
	if req.TaskType == "" {
		return dispatcher.ErrInvalidInput
	}
	if req.Attempt <= 0 {
		req.Attempt = 1
	}
	if req.MaxAttempts <= 0 {
		req.MaxAttempts = opts.DefaultMaxAttempts
	}
	if req.EmittedAt.IsZero() {
		req.EmittedAt = time.Now().UTC()
	}
	return nil
}
