package usecase

import (
	"context"

	"ingest-srv/internal/datasource"
	repo "ingest-srv/internal/datasource/repository"
)

// UpdateTarget validates and applies changes to a crawl target.
func (uc *implUseCase) UpdateTarget(ctx context.Context, input datasource.UpdateTargetInput) (datasource.UpdateTargetOutput, error) {
	if input.ID == "" {
		return datasource.UpdateTargetOutput{}, datasource.ErrTargetNotFound
	}

	// Validate interval if provided.
	if input.CrawlIntervalMinutes != nil && *input.CrawlIntervalMinutes <= 0 {
		uc.l.Warnf(ctx, "datasource.usecase.UpdateTarget: crawl_interval_minutes must be > 0")
		return datasource.UpdateTargetOutput{}, datasource.ErrTargetUpdateFailed
	}

	opt := repo.UpdateTargetOptions{
		ID:                   input.ID,
		Value:                input.Value,
		Label:                input.Label,
		PlatformMeta:         input.PlatformMeta,
		IsActive:             input.IsActive,
		Priority:             input.Priority,
		CrawlIntervalMinutes: input.CrawlIntervalMinutes,
	}

	result, err := uc.repo.UpdateTarget(ctx, opt)
	if err != nil {
		uc.l.Errorf(ctx, "datasource.usecase.UpdateTarget.repo.UpdateTarget: id=%s err=%v", input.ID, err)
		return datasource.UpdateTargetOutput{}, datasource.ErrTargetUpdateFailed
	}

	return datasource.UpdateTargetOutput{Target: result}, nil
}
