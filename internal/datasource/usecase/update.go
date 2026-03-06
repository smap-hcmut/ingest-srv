package usecase

import (
	"context"

	"ingest-srv/internal/datasource"
	repo "ingest-srv/internal/datasource/repository"
	"ingest-srv/internal/model"
)

// Update validates input, enforces state guards, and updates a data source.
func (uc *implUseCase) Update(ctx context.Context, input datasource.UpdateInput) (datasource.UpdateOutput, error) {
	if input.ID == "" {
		uc.l.Warnf(ctx, "datasource.usecase.Update: empty id")
		return datasource.UpdateOutput{}, datasource.ErrNotFound
	}

	// Fetch current state for business validation
	current, err := uc.repo.DetailDataSource(ctx, input.ID)
	if err != nil {
		uc.l.Errorf(ctx, "datasource.usecase.Update.repo.DetailDataSource: id=%s err=%v", input.ID, err)
		return datasource.UpdateOutput{}, datasource.ErrNotFound
	}
	if current.ID == "" {
		uc.l.Warnf(ctx, "datasource.usecase.Update: not found id=%s", input.ID)
		return datasource.UpdateOutput{}, datasource.ErrNotFound
	}

	// State guard: config/mapping changes not allowed on ACTIVE source
	hasRuntimeChange := len(input.Config) > 0 || len(input.MappingRules) > 0
	if hasRuntimeChange && current.Status == model.SourceStatusActive {
		uc.l.Warnf(ctx, "datasource.usecase.Update: cannot update config/mapping on ACTIVE source id=%s", input.ID)
		return datasource.UpdateOutput{}, datasource.ErrUpdateNotAllowed
	}

	// Build update options — only non-empty fields
	opt := repo.UpdateDataSourceOptions{
		ID:           input.ID,
		Name:         input.Name,
		Description:  input.Description,
		Config:       input.Config,
		AccountRef:   input.AccountRef,
		MappingRules: input.MappingRules,
	}

	// If config/mapping changed, reset dryrun state so source must revalidate
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
