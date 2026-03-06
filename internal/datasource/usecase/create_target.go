package usecase

import (
	"context"

	"ingest-srv/internal/datasource"
	repo "ingest-srv/internal/datasource/repository"
	"ingest-srv/internal/model"
)

// CreateTarget validates input, checks source is CRAWL, and creates a new crawl target.
func (uc *implUseCase) CreateTarget(ctx context.Context, input datasource.CreateTargetInput) (datasource.CreateTargetOutput, error) {
	if input.DataSourceID == "" {
		uc.l.Warnf(ctx, "datasource.usecase.CreateTarget: data_source_id is required")
		return datasource.CreateTargetOutput{}, datasource.ErrProjectIDRequired
	}
	if input.Value == "" {
		uc.l.Warnf(ctx, "datasource.usecase.CreateTarget: value is required")
		return datasource.CreateTargetOutput{}, datasource.ErrTargetValueRequired
	}
	if err := uc.validateTargetType(input.TargetType); err != nil {
		return datasource.CreateTargetOutput{}, err
	}

	// Verify the parent data source exists and is CRAWL.
	ds, err := uc.repo.DetailDataSource(ctx, input.DataSourceID)
	if err != nil {
		uc.l.Errorf(ctx, "datasource.usecase.CreateTarget.DetailDataSource: %v", err)
		return datasource.CreateTargetOutput{}, datasource.ErrTargetCreateFailed
	}
	if ds.ID == "" {
		return datasource.CreateTargetOutput{}, datasource.ErrNotFound
	}
	if ds.SourceCategory != model.SourceCategoryCrawl {
		return datasource.CreateTargetOutput{}, datasource.ErrSourceNotCrawl
	}

	// Inherit crawl_interval_minutes from datasource if not provided.
	interval := input.CrawlIntervalMinutes
	if interval <= 0 && ds.CrawlIntervalMinutes != nil && *ds.CrawlIntervalMinutes > 0 {
		interval = *ds.CrawlIntervalMinutes
	}

	opt := repo.CreateTargetOptions{
		DataSourceID:         input.DataSourceID,
		TargetType:           input.TargetType,
		Value:                input.Value,
		Label:                input.Label,
		PlatformMeta:         input.PlatformMeta,
		IsActive:             input.IsActive,
		Priority:             input.Priority,
		CrawlIntervalMinutes: interval,
	}

	result, err := uc.repo.CreateTarget(ctx, opt)
	if err != nil {
		uc.l.Errorf(ctx, "datasource.usecase.CreateTarget.repo.CreateTarget: %v", err)
		return datasource.CreateTargetOutput{}, datasource.ErrTargetCreateFailed
	}

	return datasource.CreateTargetOutput{Target: result}, nil
}
