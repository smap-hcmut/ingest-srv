package usecase

import (
	"smap-collector/internal/webhook"
	"smap-collector/pkg/log"
	"smap-collector/pkg/project"
)

type implUseCase struct {
	l             log.Logger
	projectClient project.Client
}

// NewUseCase tạo webhook usecase mới.
func NewUseCase(l log.Logger, projectClient project.Client) webhook.UseCase {
	return &implUseCase{
		l:             l,
		projectClient: projectClient,
	}
}
