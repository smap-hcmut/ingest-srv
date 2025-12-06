package usecase

import (
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
}

// NewUseCase creates a new dispatcher usecase (legacy mode without state/webhook).
func NewUseCase(l log.Logger, prod producer.Producer, opts dispatcher.Options) dispatcher.UseCase {
	return NewUseCaseWithDeps(l, prod, opts, nil, nil)
}

// NewUseCaseWithDeps creates a new dispatcher usecase with optional state and webhook dependencies.
func NewUseCaseWithDeps(
	l log.Logger,
	prod producer.Producer,
	opts dispatcher.Options,
	stateUC state.UseCase,
	webhookUC webhook.UseCase,
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

	return &implUseCase{
		l:              l,
		prod:           prod,
		defaultOptions: opts,
		stateUC:        stateUC,
		webhookUC:      webhookUC,
	}
}
