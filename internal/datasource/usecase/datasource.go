package usecase

import (
	"context"
	"strings"

	"ingest-srv/internal/datasource"
	repo "ingest-srv/internal/datasource/repository"
	"ingest-srv/internal/model"

	"github.com/smap-hcmut/shared-libs/go/auth"
)

// Create validates input, enforces business rules, and creates a new data source.
func (uc *implUseCase) Create(ctx context.Context, input datasource.CreateInput) (datasource.CreateOutput, error) {
	if err := uc.validCreateInput(input); err != nil {
		uc.l.Warnf(ctx, "datasource.usecase.Create.validCreateInput: %v", err)
		return datasource.CreateOutput{}, err
	}

	userID, _ := auth.GetUserIDFromContext(ctx)

	opt := repo.CreateDataSourceOptions{
		ProjectID:              strings.TrimSpace(input.ProjectID),
		Name:                   strings.TrimSpace(input.Name),
		Description:            input.Description,
		SourceType:             strings.TrimSpace(input.SourceType),
		SourceCategory:         strings.TrimSpace(input.SourceCategory),
		Config:                 input.Config,
		AccountRef:             input.AccountRef,
		MappingRules:           input.MappingRules,
		CrawlMode:              strings.TrimSpace(input.CrawlMode),
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

// Detail fetches a single data source by ID.
func (uc *implUseCase) Detail(ctx context.Context, id string) (datasource.DetailOutput, error) {
	if err := uc.validDetailInput(id); err != nil {
		uc.l.Warnf(ctx, "datasource.usecase.Detail.validDetailInput: %v", err)
		return datasource.DetailOutput{}, err
	}

	result, err := uc.repo.DetailDataSource(ctx, strings.TrimSpace(id))
	if err != nil {
		uc.l.Errorf(ctx, "datasource.usecase.Detail.repo.DetailDataSource: id=%s err=%v", id, err)
		return datasource.DetailOutput{}, datasource.ErrNotFound
	}

	if result.ID == "" {
		uc.l.Warnf(ctx, "datasource.usecase.Detail: not found id=%s", id)
		return datasource.DetailOutput{}, datasource.ErrNotFound
	}

	return datasource.DetailOutput{DataSource: result}, nil
}

// List fetches data sources with pagination and filters.
func (uc *implUseCase) List(ctx context.Context, input datasource.ListInput) (datasource.ListOutput, error) {
	if err := uc.validListInput(input); err != nil {
		uc.l.Warnf(ctx, "datasource.usecase.List.validListInput: %v", err)
		return datasource.ListOutput{}, err
	}

	input.Paginator.Adjust()

	opt := repo.GetDataSourcesOptions{
		ProjectID:      strings.TrimSpace(input.ProjectID),
		Status:         strings.TrimSpace(input.Status),
		SourceType:     strings.TrimSpace(input.SourceType),
		SourceCategory: strings.TrimSpace(input.SourceCategory),
		CrawlMode:      strings.TrimSpace(input.CrawlMode),
		Name:           strings.TrimSpace(input.Name),
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

// Update validates input, enforces state guards, and updates a data source.
func (uc *implUseCase) Update(ctx context.Context, input datasource.UpdateInput) (datasource.UpdateOutput, error) {
	if err := uc.validUpdateInput(input); err != nil {
		uc.l.Warnf(ctx, "datasource.usecase.Update.validUpdateInput: %v", err)
		return datasource.UpdateOutput{}, err
	}

	current, err := uc.repo.DetailDataSource(ctx, strings.TrimSpace(input.ID))
	if err != nil {
		uc.l.Errorf(ctx, "datasource.usecase.Update.repo.DetailDataSource: id=%s err=%v", input.ID, err)
		return datasource.UpdateOutput{}, datasource.ErrNotFound
	}
	if current.ID == "" {
		uc.l.Warnf(ctx, "datasource.usecase.Update: not found id=%s", input.ID)
		return datasource.UpdateOutput{}, datasource.ErrNotFound
	}

	hasRuntimeChange := len(input.Config) > 0 || len(input.MappingRules) > 0
	if hasRuntimeChange && current.Status == model.SourceStatusActive {
		uc.l.Warnf(ctx, "datasource.usecase.Update: cannot update config/mapping on ACTIVE source id=%s", input.ID)
		return datasource.UpdateOutput{}, datasource.ErrUpdateNotAllowed
	}

	opt := repo.UpdateDataSourceOptions{
		ID:           strings.TrimSpace(input.ID),
		Name:         strings.TrimSpace(input.Name),
		Description:  input.Description,
		Config:       input.Config,
		AccountRef:   input.AccountRef,
		MappingRules: input.MappingRules,
	}

	if hasRuntimeChange {
		opt.DryrunStatus = string(model.DryrunStatusNotRequired)
		opt.DryrunLastResultID = ""
	}

	result, err := uc.repo.UpdateDataSource(ctx, opt)
	if err != nil {
		uc.l.Errorf(ctx, "datasource.usecase.Update.repo.UpdateDataSource: id=%s err=%v", input.ID, err)
		return datasource.UpdateOutput{}, datasource.ErrUpdateFailed
	}
	if result.ID == "" {
		uc.l.Warnf(ctx, "datasource.usecase.Update: not found after update id=%s", input.ID)
		return datasource.UpdateOutput{}, datasource.ErrNotFound
	}

	return datasource.UpdateOutput{DataSource: result}, nil
}

// Archive moves a data source into ARCHIVED while keeping it queryable.
func (uc *implUseCase) Archive(ctx context.Context, id string) error {
	if err := uc.validArchiveInput(id); err != nil {
		uc.l.Warnf(ctx, "datasource.usecase.Archive.validArchiveInput: %v", err)
		return err
	}

	current, err := uc.repo.DetailDataSource(ctx, strings.TrimSpace(id))
	if err != nil {
		uc.l.Errorf(ctx, "datasource.usecase.Archive.repo.DetailDataSource: id=%s err=%v", id, err)
		return datasource.ErrNotFound
	}
	if current.ID == "" {
		return datasource.ErrNotFound
	}
	if current.Status == model.SourceStatusArchived {
		return nil
	}

	if err := uc.repo.ArchiveDataSource(ctx, strings.TrimSpace(id)); err != nil {
		uc.l.Errorf(ctx, "datasource.usecase.Archive.repo.ArchiveDataSource: id=%s err=%v", id, err)
		if err == repo.ErrFailedToGet {
			return datasource.ErrNotFound
		}
		return datasource.ErrDeleteFailed
	}

	return nil
}

// Delete soft-deletes a datasource only after it has been archived.
func (uc *implUseCase) Delete(ctx context.Context, id string) error {
	if err := uc.validArchiveInput(id); err != nil {
		uc.l.Warnf(ctx, "datasource.usecase.Delete.validArchiveInput: %v", err)
		return err
	}

	current, err := uc.repo.DetailDataSource(ctx, strings.TrimSpace(id))
	if err != nil {
		uc.l.Errorf(ctx, "datasource.usecase.Delete.repo.DetailDataSource: id=%s err=%v", id, err)
		return datasource.ErrNotFound
	}
	if current.ID == "" {
		return datasource.ErrNotFound
	}
	if current.Status != model.SourceStatusArchived {
		return datasource.ErrDeleteRequiresArchived
	}

	if err := uc.repo.DeleteDataSource(ctx, strings.TrimSpace(id)); err != nil {
		uc.l.Errorf(ctx, "datasource.usecase.Delete.repo.DeleteDataSource: id=%s err=%v", id, err)
		if err == repo.ErrFailedToGet {
			return datasource.ErrNotFound
		}
		return datasource.ErrDeleteFailed
	}

	return nil
}

func (uc *implUseCase) validCreateInput(input datasource.CreateInput) error {
	if strings.TrimSpace(input.ProjectID) == "" {
		return datasource.ErrProjectIDRequired
	}
	if strings.TrimSpace(input.Name) == "" {
		return datasource.ErrNameRequired
	}
	if strings.TrimSpace(input.SourceType) == "" {
		return datasource.ErrSourceTypeRequired
	}
	if err := uc.validateSourceType(strings.TrimSpace(input.SourceType)); err != nil {
		return err
	}

	category := strings.TrimSpace(input.SourceCategory)
	if category == "" {
		category = uc.inferCategory(strings.TrimSpace(input.SourceType))
	}
	if err := uc.validateSourceCategory(category); err != nil {
		return err
	}

	if category == string(model.SourceCategoryCrawl) {
		if strings.TrimSpace(input.CrawlMode) == "" || input.CrawlIntervalMinutes <= 0 {
			return datasource.ErrCrawlConfigRequired
		}
	}
	if strings.TrimSpace(input.CrawlMode) != "" {
		if err := uc.validateCrawlMode(strings.TrimSpace(input.CrawlMode)); err != nil {
			return err
		}
	}

	return nil
}

func (uc *implUseCase) validDetailInput(id string) error {
	if strings.TrimSpace(id) == "" {
		return datasource.ErrNotFound
	}
	return nil
}

func (uc *implUseCase) validListInput(input datasource.ListInput) error {
	if strings.TrimSpace(input.SourceType) != "" {
		if err := uc.validateSourceType(strings.TrimSpace(input.SourceType)); err != nil {
			return err
		}
	}
	if strings.TrimSpace(input.SourceCategory) != "" {
		if err := uc.validateSourceCategory(strings.TrimSpace(input.SourceCategory)); err != nil {
			return err
		}
	}
	if strings.TrimSpace(input.CrawlMode) != "" {
		if err := uc.validateCrawlMode(strings.TrimSpace(input.CrawlMode)); err != nil {
			return err
		}
	}
	return nil
}

func (uc *implUseCase) validUpdateInput(input datasource.UpdateInput) error {
	if strings.TrimSpace(input.ID) == "" {
		return datasource.ErrNotFound
	}
	return nil
}

func (uc *implUseCase) validArchiveInput(id string) error {
	if strings.TrimSpace(id) == "" {
		return datasource.ErrNotFound
	}
	return nil
}
