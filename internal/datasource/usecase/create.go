package usecase

import (
	"context"

	"ingest-srv/internal/datasource"
	repo "ingest-srv/internal/datasource/repository"
	"ingest-srv/pkg/scope"
)

// Create validates input, enforces business rules, and creates a new data source.
func (uc *implUseCase) Create(ctx context.Context, input datasource.CreateInput) (datasource.CreateOutput, error) {
	// --- Business validation ---
	if input.ProjectID == "" {
		uc.l.Warnf(ctx, "datasource.usecase.Create: project_id is required")
		return datasource.CreateOutput{}, datasource.ErrProjectIDRequired
	}
	if input.Name == "" {
		uc.l.Warnf(ctx, "datasource.usecase.Create: name is required")
		return datasource.CreateOutput{}, datasource.ErrNameRequired
	}
	if input.SourceType == "" {
		uc.l.Warnf(ctx, "datasource.usecase.Create: source_type is required")
		return datasource.CreateOutput{}, datasource.ErrSourceTypeRequired
	}
	if err := uc.validateSourceType(input.SourceType); err != nil {
		uc.l.Warnf(ctx, "datasource.usecase.Create: invalid source_type=%s", input.SourceType)
		return datasource.CreateOutput{}, err
	}
	if input.SourceCategory == "" {
		input.SourceCategory = uc.inferCategory(input.SourceType)
	}
	if err := uc.validateSourceCategory(input.SourceCategory); err != nil {
		uc.l.Warnf(ctx, "datasource.usecase.Create: invalid source_category=%s", input.SourceCategory)
		return datasource.CreateOutput{}, err
	}

	// CRAWL source requires crawl config
	if input.SourceCategory == "CRAWL" {
		if input.CrawlMode == "" || input.CrawlIntervalMinutes <= 0 {
			uc.l.Warnf(ctx, "datasource.usecase.Create: crawl source requires crawl_mode and interval")
			return datasource.CreateOutput{}, datasource.ErrCrawlConfigRequired
		}
	}
	if input.CrawlMode != "" {
		if err := uc.validateCrawlMode(input.CrawlMode); err != nil {
			uc.l.Warnf(ctx, "datasource.usecase.Create: invalid crawl_mode=%s", input.CrawlMode)
			return datasource.CreateOutput{}, err
		}
	}

	// Get user from context
	userID, _ := scope.GetUserIDFromContext(ctx)

	// Convert Input → Options
	opt := repo.CreateDataSourceOptions{
		ProjectID:              input.ProjectID,
		Name:                   input.Name,
		Description:            input.Description,
		SourceType:             input.SourceType,
		SourceCategory:         input.SourceCategory,
		Config:                 input.Config,
		AccountRef:             input.AccountRef,
		MappingRules:           input.MappingRules,
		CrawlMode:              input.CrawlMode,
		CrawlIntervalMinutes:   input.CrawlIntervalMinutes,
		WebhookID:              input.WebhookID,
		WebhookSecretEncrypted: input.WebhookSecretEncrypted,
		CreatedBy:              userID,
	}

	result, err := uc.repo.CreateDataSource(ctx, opt)
	if err != nil {
		uc.l.Errorf(ctx, "datasource.usecase.Create.repo.CreateDataSource: %v", err)
		return datasource.CreateOutput{}, datasource.ErrCreateFailed
	}

	return datasource.CreateOutput{DataSource: result}, nil
}
