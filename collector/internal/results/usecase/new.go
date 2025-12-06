package usecase

import (
	"smap-collector/internal/results"
	"smap-collector/internal/state"
	"smap-collector/internal/webhook"
	"smap-collector/pkg/log"
	"smap-collector/pkg/project"
)

type implUseCase struct {
	l             log.Logger
	projectClient project.Client
	stateUC       state.UseCase
	webhookUC     webhook.UseCase
}

// NewUseCase creates a new results usecase.
// Note: DataCollected event publishing is handled by Crawler services, not Collector.
func NewUseCase(l log.Logger, projectClient project.Client, stateUC state.UseCase, webhookUC webhook.UseCase) results.UseCase {
	return &implUseCase{
		l:             l,
		projectClient: projectClient,
		stateUC:       stateUC,
		webhookUC:     webhookUC,
	}
}
