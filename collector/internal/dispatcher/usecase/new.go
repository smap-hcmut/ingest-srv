package usecase

import (
	"smap-collector/config"
	"smap-collector/internal/dispatcher"
	"smap-collector/internal/dispatcher/delivery/rabbitmq/producer"
	"smap-collector/internal/models"
	"smap-collector/internal/state"
	"smap-collector/internal/webhook"
	"smap-collector/pkg/log"
)

type implUseCase struct {
	l              log.Logger
	prod           producer.Producer
	defaultOptions dispatcher.Options

	// Optional dependencies for event-driven architecture
	stateUC   state.UseCase
	webhookUC webhook.UseCase

	// Crawl limits configuration (for config-driven limits)
	crawlLimitsCfg config.CrawlLimitsConfig
}

// NewUseCase creates a new dispatcher usecase (legacy mode without state/webhook).
func NewUseCase(l log.Logger, prod producer.Producer, opts dispatcher.Options) dispatcher.UseCase {
	return NewUseCaseWithDeps(l, prod, opts, nil, nil, config.CrawlLimitsConfig{})
}

// NewUseCaseWithDeps creates a new dispatcher usecase with optional state and webhook dependencies.
func NewUseCaseWithDeps(
	l log.Logger,
	prod producer.Producer,
	opts dispatcher.Options,
	stateUC state.UseCase,
	webhookUC webhook.UseCase,
	crawlLimitsCfg config.CrawlLimitsConfig,
) dispatcher.UseCase {
	if l == nil || prod == nil {
		return nil
	}

	if opts.DefaultMaxAttempts <= 0 {
		opts.DefaultMaxAttempts = 3
	}
	if opts.SchemaVersion <= 0 {
		opts.SchemaVersion = 1
	}
	if len(opts.PlatformQueues) == 0 {
		opts.PlatformQueues = map[models.Platform]string{
			models.PlatformYouTube: "crawler.youtube.queue",
			models.PlatformTikTok:  "crawler.tiktok.queue",
		}
	}

	// Set default crawl limits if not provided
	if crawlLimitsCfg.DefaultLimitPerKeyword == 0 {
		crawlLimitsCfg.DefaultLimitPerKeyword = 50
	}
	if crawlLimitsCfg.DefaultMaxComments == 0 {
		crawlLimitsCfg.DefaultMaxComments = 100
	}
	if crawlLimitsCfg.DefaultMaxAttempts == 0 {
		crawlLimitsCfg.DefaultMaxAttempts = 3
	}
	if crawlLimitsCfg.DryRunLimitPerKeyword == 0 {
		crawlLimitsCfg.DryRunLimitPerKeyword = 3
	}
	if crawlLimitsCfg.DryRunMaxComments == 0 {
		crawlLimitsCfg.DryRunMaxComments = 5
	}

	return &implUseCase{
		l:              l,
		prod:           prod,
		defaultOptions: opts,
		stateUC:        stateUC,
		webhookUC:      webhookUC,
		crawlLimitsCfg: crawlLimitsCfg,
	}
}
