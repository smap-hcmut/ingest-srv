package usecase

import (
	"smap-collector/internal/state"
	"smap-collector/internal/state/repository"
	"smap-collector/pkg/log"
)

type implUseCase struct {
	l    log.Logger
	repo repository.StateRepository
	opts state.Options
}

// NewUseCase tạo state usecase mới.
func NewUseCase(l log.Logger, repo repository.StateRepository, opts state.Options) state.UseCase {
	if opts.TTL == 0 {
		opts.TTL = state.DefaultTTL
	}

	return &implUseCase{
		l:    l,
		repo: repo,
		opts: opts,
	}
}
