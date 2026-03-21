package usecase

import (
	"context"
	"strings"

	"ingest-srv/internal/datasource"
	repo "ingest-srv/internal/datasource/repository"
	"ingest-srv/internal/model"
)

// CreateKeywordTarget validates input, checks source is CRAWL, and creates a grouped keyword target.
func (uc *implUseCase) CreateKeywordTarget(ctx context.Context, input datasource.CreateTargetGroupInput) (datasource.CreateTargetOutput, error) {
	return uc.createTarget(ctx, input, model.TargetTypeKeyword)
}

// CreateProfileTarget validates input, checks source is CRAWL, and creates a grouped profile target.
func (uc *implUseCase) CreateProfileTarget(ctx context.Context, input datasource.CreateTargetGroupInput) (datasource.CreateTargetOutput, error) {
	return uc.createTarget(ctx, input, model.TargetTypeProfile)
}

// CreatePostTarget validates input, checks source is CRAWL, and creates a grouped post target.
func (uc *implUseCase) CreatePostTarget(ctx context.Context, input datasource.CreateTargetGroupInput) (datasource.CreateTargetOutput, error) {
	return uc.createTarget(ctx, input, model.TargetTypePostURL)
}

func (uc *implUseCase) createTarget(ctx context.Context, input datasource.CreateTargetGroupInput, targetType model.TargetType) (datasource.CreateTargetOutput, error) {
	if err := uc.validCreateTargetGroupInput(input); err != nil {
		uc.l.Warnf(ctx, "datasource.usecase.createTarget.validCreateTargetGroupInput: %v", err)
		return datasource.CreateTargetOutput{}, err
	}

	ds, err := uc.repo.DetailDataSource(ctx, strings.TrimSpace(input.DataSourceID))
	if err != nil {
		uc.l.Errorf(ctx, "datasource.usecase.createTarget.DetailDataSource: %v", err)
		return datasource.CreateTargetOutput{}, datasource.ErrTargetCreateFailed
	}
	if ds.ID == "" {
		return datasource.CreateTargetOutput{}, datasource.ErrNotFound
	}
	if err := uc.ensureDataSourceNotArchived(ds); err != nil {
		return datasource.CreateTargetOutput{}, err
	}
	if ds.SourceCategory != model.SourceCategoryCrawl {
		return datasource.CreateTargetOutput{}, datasource.ErrSourceNotCrawl
	}

	values, err := uc.prepareTargetValues(targetType, input.Values)
	if err != nil {
		return datasource.CreateTargetOutput{}, err
	}

	opt := repo.CreateTargetOptions{
		DataSourceID:         strings.TrimSpace(input.DataSourceID),
		TargetType:           string(targetType),
		Values:               model.TypesJSONFromStringSlice(values),
		Label:                input.Label,
		PlatformMeta:         input.PlatformMeta,
		IsActive:             false,
		Priority:             input.Priority,
		CrawlIntervalMinutes: input.CrawlIntervalMinutes,
	}

	result, err := uc.repo.CreateTarget(ctx, opt)
	if err != nil {
		uc.l.Errorf(ctx, "datasource.usecase.createTarget.repo.CreateTarget: %v", err)
		return datasource.CreateTargetOutput{}, datasource.ErrTargetCreateFailed
	}

	return datasource.CreateTargetOutput{Target: result}, nil
}

// DetailTarget fetches a single crawl target by ID with datasource ownership check.
func (uc *implUseCase) DetailTarget(ctx context.Context, input datasource.DetailTargetInput) (datasource.DetailTargetOutput, error) {
	if err := uc.validDetailTargetInput(input); err != nil {
		uc.l.Warnf(ctx, "datasource.usecase.DetailTarget.validDetailTargetInput: %v", err)
		return datasource.DetailTargetOutput{}, err
	}

	result, err := uc.repo.GetTarget(ctx, repo.GetTargetOptions{
		DataSourceID: strings.TrimSpace(input.DataSourceID),
		ID:           strings.TrimSpace(input.ID),
	})
	if err != nil {
		uc.l.Errorf(ctx, "datasource.usecase.DetailTarget.repo.GetTarget: id=%s err=%v", input.ID, err)
		return datasource.DetailTargetOutput{}, datasource.ErrTargetNotFound
	}

	return datasource.DetailTargetOutput{Target: result}, nil
}

// ListTargets returns all crawl targets for a data source.
func (uc *implUseCase) ListTargets(ctx context.Context, input datasource.ListTargetsInput) (datasource.ListTargetsOutput, error) {
	if err := uc.validListTargetsInput(input); err != nil {
		uc.l.Warnf(ctx, "datasource.usecase.ListTargets.validListTargetsInput: %v", err)
		return datasource.ListTargetsOutput{}, err
	}

	opt := repo.ListTargetsOptions{
		DataSourceID: strings.TrimSpace(input.DataSourceID),
		TargetType:   strings.TrimSpace(input.TargetType),
		IsActive:     input.IsActive,
	}

	results, err := uc.repo.ListTargets(ctx, opt)
	if err != nil {
		uc.l.Errorf(ctx, "datasource.usecase.ListTargets.repo.ListTargets: %v", err)
		return datasource.ListTargetsOutput{}, datasource.ErrTargetListFailed
	}

	return datasource.ListTargetsOutput{Targets: results}, nil
}

// UpdateTarget validates and applies changes to a crawl target.
func (uc *implUseCase) UpdateTarget(ctx context.Context, input datasource.UpdateTargetInput) (datasource.UpdateTargetOutput, error) {
	if err := uc.validUpdateTargetInput(input); err != nil {
		uc.l.Warnf(ctx, "datasource.usecase.UpdateTarget.validUpdateTargetInput: %v", err)
		return datasource.UpdateTargetOutput{}, err
	}

	current, err := uc.repo.GetTarget(ctx, repo.GetTargetOptions{
		DataSourceID: strings.TrimSpace(input.DataSourceID),
		ID:           strings.TrimSpace(input.ID),
	})
	if err != nil {
		uc.l.Errorf(ctx, "datasource.usecase.UpdateTarget.repo.GetTarget: id=%s err=%v", input.ID, err)
		if err == repo.ErrTargetNotFound {
			return datasource.UpdateTargetOutput{}, datasource.ErrTargetNotFound
		}
		return datasource.UpdateTargetOutput{}, datasource.ErrTargetUpdateFailed
	}
	if err := uc.ensureTargetDryrunNotRunning(ctx, current.ID); err != nil {
		uc.l.Warnf(ctx, "datasource.usecase.UpdateTarget.ensureTargetDryrunNotRunning: target_id=%s err=%v", current.ID, err)
		return datasource.UpdateTargetOutput{}, err
	}

	var values []string
	valuesForComparison := current.Values
	if input.Values != nil {
		values, err = uc.prepareTargetValues(current.TargetType, input.Values)
		if err != nil {
			return datasource.UpdateTargetOutput{}, err
		}
		valuesForComparison = values
	}
	materialChanged := uc.hasMaterialTargetChange(current, valuesForComparison, input)

	opt := repo.UpdateTargetOptions{
		DataSourceID:         strings.TrimSpace(input.DataSourceID),
		ID:                   strings.TrimSpace(input.ID),
		Label:                input.Label,
		PlatformMeta:         input.PlatformMeta,
		Priority:             input.Priority,
		CrawlIntervalMinutes: input.CrawlIntervalMinutes,
	}
	if input.Values != nil {
		opt.Values = model.TypesJSONFromStringSlice(values)
	}
	if materialChanged && current.IsActive {
		disabled := false
		opt.IsActive = &disabled
	}

	result, err := uc.repo.UpdateTarget(ctx, opt)
	if err != nil {
		uc.l.Errorf(ctx, "datasource.usecase.UpdateTarget.repo.UpdateTarget: id=%s err=%v", input.ID, err)
		if err == repo.ErrTargetNotFound {
			return datasource.UpdateTargetOutput{}, datasource.ErrTargetNotFound
		}
		return datasource.UpdateTargetOutput{}, datasource.ErrTargetUpdateFailed
	}
	if materialChanged {
		if err := uc.markDatasourcePendingAfterMaterialTargetChange(ctx, strings.TrimSpace(input.DataSourceID)); err != nil {
			uc.l.Errorf(ctx, "datasource.usecase.UpdateTarget.markDatasourcePendingAfterMaterialTargetChange: source_id=%s target_id=%s err=%v", input.DataSourceID, input.ID, err)
			return datasource.UpdateTargetOutput{}, err
		}
	}

	return datasource.UpdateTargetOutput{Target: result}, nil
}

// ActivateTarget enables one crawl target after dryrun evidence is available.
func (uc *implUseCase) ActivateTarget(ctx context.Context, input datasource.ActivateTargetInput) (datasource.ActivateTargetOutput, error) {
	if err := uc.validActivateTargetInput(input); err != nil {
		uc.l.Warnf(ctx, "datasource.usecase.ActivateTarget.validActivateTargetInput: %v", err)
		return datasource.ActivateTargetOutput{}, err
	}

	current, err := uc.repo.GetTarget(ctx, repo.GetTargetOptions{
		DataSourceID: strings.TrimSpace(input.DataSourceID),
		ID:           strings.TrimSpace(input.ID),
	})
	if err != nil {
		uc.l.Errorf(ctx, "datasource.usecase.ActivateTarget.repo.GetTarget: id=%s err=%v", input.ID, err)
		if err == repo.ErrTargetNotFound {
			return datasource.ActivateTargetOutput{}, datasource.ErrTargetNotFound
		}
		return datasource.ActivateTargetOutput{}, datasource.ErrTargetUpdateFailed
	}
	if err := uc.ensureTargetDryrunNotRunning(ctx, current.ID); err != nil {
		uc.l.Warnf(ctx, "datasource.usecase.ActivateTarget.ensureTargetDryrunNotRunning: target_id=%s err=%v", current.ID, err)
		return datasource.ActivateTargetOutput{}, err
	}

	if current.IsActive {
		return datasource.ActivateTargetOutput{Target: current}, nil
	}

	latest, err := uc.repo.GetLatestDryrunByTarget(ctx, current.ID)
	if err != nil {
		uc.l.Errorf(ctx, "datasource.usecase.ActivateTarget.repo.GetLatestDryrunByTarget: target_id=%s err=%v", current.ID, err)
		return datasource.ActivateTargetOutput{}, datasource.ErrTargetUpdateFailed
	}
	if latest.ID == "" || !model.IsUsableDryrunStatus(latest.Status) {
		return datasource.ActivateTargetOutput{}, datasource.ErrTargetActivateNotAllowed
	}

	enabled := true
	updated, err := uc.repo.UpdateTarget(ctx, repo.UpdateTargetOptions{
		DataSourceID: strings.TrimSpace(input.DataSourceID),
		ID:           strings.TrimSpace(input.ID),
		IsActive:     &enabled,
	})
	if err != nil {
		uc.l.Errorf(ctx, "datasource.usecase.ActivateTarget.repo.UpdateTarget: id=%s err=%v", input.ID, err)
		if err == repo.ErrTargetNotFound {
			return datasource.ActivateTargetOutput{}, datasource.ErrTargetNotFound
		}
		return datasource.ActivateTargetOutput{}, datasource.ErrTargetUpdateFailed
	}

	return datasource.ActivateTargetOutput{Target: updated}, nil
}

// DeactivateTarget disables one crawl target.
func (uc *implUseCase) DeactivateTarget(ctx context.Context, input datasource.DeactivateTargetInput) (datasource.DeactivateTargetOutput, error) {
	if err := uc.validDeactivateTargetInput(input); err != nil {
		uc.l.Warnf(ctx, "datasource.usecase.DeactivateTarget.validDeactivateTargetInput: %v", err)
		return datasource.DeactivateTargetOutput{}, err
	}

	current, err := uc.repo.GetTarget(ctx, repo.GetTargetOptions{
		DataSourceID: strings.TrimSpace(input.DataSourceID),
		ID:           strings.TrimSpace(input.ID),
	})
	if err != nil {
		uc.l.Errorf(ctx, "datasource.usecase.DeactivateTarget.repo.GetTarget: id=%s err=%v", input.ID, err)
		if err == repo.ErrTargetNotFound {
			return datasource.DeactivateTargetOutput{}, datasource.ErrTargetNotFound
		}
		return datasource.DeactivateTargetOutput{}, datasource.ErrTargetUpdateFailed
	}
	if err := uc.ensureTargetDryrunNotRunning(ctx, current.ID); err != nil {
		uc.l.Warnf(ctx, "datasource.usecase.DeactivateTarget.ensureTargetDryrunNotRunning: target_id=%s err=%v", current.ID, err)
		return datasource.DeactivateTargetOutput{}, err
	}

	if !current.IsActive {
		return datasource.DeactivateTargetOutput{Target: current}, nil
	}

	if err := uc.ensureCanRemoveActiveTarget(ctx, input.DataSourceID, current.IsActive, datasource.ErrTargetDeactivateNotAllowed); err != nil {
		if err == datasource.ErrNotFound || err == datasource.ErrTargetDeactivateNotAllowed {
			return datasource.DeactivateTargetOutput{}, err
		}
		return datasource.DeactivateTargetOutput{}, datasource.ErrTargetUpdateFailed
	}

	disabled := false
	updated, err := uc.repo.UpdateTarget(ctx, repo.UpdateTargetOptions{
		DataSourceID: strings.TrimSpace(input.DataSourceID),
		ID:           strings.TrimSpace(input.ID),
		IsActive:     &disabled,
	})
	if err != nil {
		uc.l.Errorf(ctx, "datasource.usecase.DeactivateTarget.repo.UpdateTarget: id=%s err=%v", input.ID, err)
		if err == repo.ErrTargetNotFound {
			return datasource.DeactivateTargetOutput{}, datasource.ErrTargetNotFound
		}
		return datasource.DeactivateTargetOutput{}, datasource.ErrTargetUpdateFailed
	}

	return datasource.DeactivateTargetOutput{Target: updated}, nil
}

// DeleteTarget hard-deletes a crawl target by ID with datasource ownership check.
func (uc *implUseCase) DeleteTarget(ctx context.Context, input datasource.DeleteTargetInput) error {
	if err := uc.validDeleteTargetInput(input); err != nil {
		uc.l.Warnf(ctx, "datasource.usecase.DeleteTarget.validDeleteTargetInput: %v", err)
		return err
	}

	current, err := uc.repo.GetTarget(ctx, repo.GetTargetOptions{
		DataSourceID: strings.TrimSpace(input.DataSourceID),
		ID:           strings.TrimSpace(input.ID),
	})
	if err != nil {
		uc.l.Errorf(ctx, "datasource.usecase.DeleteTarget.repo.GetTarget: id=%s err=%v", input.ID, err)
		if err == repo.ErrTargetNotFound {
			return datasource.ErrTargetNotFound
		}
		return datasource.ErrTargetDeleteFailed
	}
	if err := uc.ensureTargetDryrunNotRunning(ctx, current.ID); err != nil {
		uc.l.Warnf(ctx, "datasource.usecase.DeleteTarget.ensureTargetDryrunNotRunning: target_id=%s err=%v", current.ID, err)
		return err
	}

	if err := uc.ensureCanRemoveActiveTarget(ctx, input.DataSourceID, current.IsActive, datasource.ErrTargetDeleteNotAllowed); err != nil {
		if err == datasource.ErrNotFound || err == datasource.ErrTargetDeleteNotAllowed {
			return err
		}
		return datasource.ErrTargetDeleteFailed
	}

	if err := uc.repo.DeleteTarget(ctx, repo.DeleteTargetOptions{
		DataSourceID: strings.TrimSpace(input.DataSourceID),
		ID:           strings.TrimSpace(input.ID),
	}); err != nil {
		uc.l.Errorf(ctx, "datasource.usecase.DeleteTarget.repo.DeleteTarget: id=%s err=%v", input.ID, err)
		if err == repo.ErrTargetNotFound {
			return datasource.ErrTargetNotFound
		}
		return datasource.ErrTargetDeleteFailed
	}

	return nil
}
