package usecase

import (
	"context"

	"ingest-srv/internal/datasource"
	repo "ingest-srv/internal/datasource/repository"
)

// List fetches data sources with pagination and filters.
func (uc *implUseCase) List(ctx context.Context, input datasource.ListInput) (datasource.ListOutput, error) {
	// Validate filter enums if provided
	if input.SourceType != "" {
		if err := uc.validateSourceType(input.SourceType); err != nil {
			uc.l.Warnf(ctx, "datasource.usecase.List: invalid source_type=%s", input.SourceType)
			return datasource.ListOutput{}, err
		}
	}
	if input.SourceCategory != "" {
		if err := uc.validateSourceCategory(input.SourceCategory); err != nil {
			uc.l.Warnf(ctx, "datasource.usecase.List: invalid source_category=%s", input.SourceCategory)
			return datasource.ListOutput{}, err
		}
	}
	if input.CrawlMode != "" {
		if err := uc.validateCrawlMode(input.CrawlMode); err != nil {
			uc.l.Warnf(ctx, "datasource.usecase.List: invalid crawl_mode=%s", input.CrawlMode)
			return datasource.ListOutput{}, err
		}
	}

	// Normalize pagination
	input.Paginator.Adjust()

	// Convert Input → Options
	opt := repo.GetDataSourcesOptions{
		ProjectID:      input.ProjectID,
		Status:         input.Status,
		SourceType:     input.SourceType,
		SourceCategory: input.SourceCategory,
		CrawlMode:      input.CrawlMode,
		Name:           input.Name,
		Paginator:      input.Paginator,
	}

	dataSources, pag, err := uc.repo.GetDataSources(ctx, opt)
	if err != nil {
		uc.l.Errorf(ctx, "datasource.usecase.List.repo.GetDataSources: %v", err)
		return datasource.ListOutput{}, datasource.ErrListFailed
	}

	return datasource.ListOutput{
		DataSources: dataSources,
		Paginator:   pag,
	}, nil
}
