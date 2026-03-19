package usecase

import (
	"context"
	"strings"

	"ingest-srv/internal/datasource"
	repo "ingest-srv/internal/datasource/repository"
	"ingest-srv/internal/model"

	"github.com/smap-hcmut/shared-libs/go/auth"
)

const (
	readinessCodeDatasourceRequired = "DATASOURCE_REQUIRED"
	readinessCodePassiveUnconfirmed = "PASSIVE_UNCONFIRMED"
	readinessCodeTargetDryrunMiss   = "TARGET_DRYRUN_MISSING"
	readinessCodeTargetDryrunFailed = "TARGET_DRYRUN_FAILED"
)

// ActivateDataSource transitions a datasource from READY to ACTIVE when runtime prerequisites are met.
func (uc *implUseCase) ActivateDataSource(ctx context.Context, id string) (datasource.ActivateOutput, error) {
	if err := uc.validDataSourceID(id); err != nil {
		uc.l.Warnf(ctx, "datasource.usecase.ActivateDataSource.validDataSourceID: %v", err)
		return datasource.ActivateOutput{}, err
	}

	current, err := uc.repo.DetailDataSource(ctx, strings.TrimSpace(id))
	if err != nil {
		uc.l.Errorf(ctx, "datasource.usecase.ActivateDataSource.repo.DetailDataSource: id=%s err=%v", id, err)
		return datasource.ActivateOutput{}, datasource.ErrNotFound
	}
	if current.ID == "" {
		return datasource.ActivateOutput{}, datasource.ErrNotFound
	}
	if current.Status != model.SourceStatusReady {
		return datasource.ActivateOutput{}, datasource.ErrActivateNotAllowed
	}

	switch current.SourceCategory {
	case model.SourceCategoryCrawl:
		if current.CrawlMode == nil || current.CrawlIntervalMinutes == nil || *current.CrawlIntervalMinutes <= 0 {
			return datasource.ActivateOutput{}, datasource.ErrActivateNotAllowed
		}

		activeTargets, err := uc.repo.CountActiveTargets(ctx, current.ID)
		if err != nil {
			uc.l.Errorf(ctx, "datasource.usecase.ActivateDataSource.repo.CountActiveTargets: id=%s err=%v", id, err)
			return datasource.ActivateOutput{}, datasource.ErrActivateNotAllowed
		}
		if activeTargets <= 0 {
			return datasource.ActivateOutput{}, datasource.ErrActivateNotAllowed
		}
	case model.SourceCategoryPassive:
		switch current.SourceType {
		case model.SourceTypeWebhook:
			if current.WebhookID == "" || current.WebhookSecretEncrypted == "" {
				return datasource.ActivateOutput{}, datasource.ErrActivateNotAllowed
			}
		default:
			return datasource.ActivateOutput{}, datasource.ErrActivateNotAllowed
		}
	default:
		return datasource.ActivateOutput{}, datasource.ErrActivateNotAllowed
	}

	updated, err := uc.repo.UpdateDataSource(ctx, repo.UpdateDataSourceOptions{
		ID:            current.ID,
		Status:        string(model.SourceStatusActive),
		ClearPausedAt: true,
	})
	if err != nil {
		uc.l.Errorf(ctx, "datasource.usecase.ActivateDataSource.repo.UpdateDataSource: id=%s err=%v", id, err)
		return datasource.ActivateOutput{}, datasource.ErrUpdateFailed
	}

	return datasource.ActivateOutput{DataSource: updated}, nil
}

// PauseDataSource transitions an ACTIVE datasource into PAUSED.
func (uc *implUseCase) PauseDataSource(ctx context.Context, id string) (datasource.PauseOutput, error) {
	if err := uc.validDataSourceID(id); err != nil {
		uc.l.Warnf(ctx, "datasource.usecase.PauseDataSource.validDataSourceID: %v", err)
		return datasource.PauseOutput{}, err
	}

	current, err := uc.repo.DetailDataSource(ctx, strings.TrimSpace(id))
	if err != nil {
		uc.l.Errorf(ctx, "datasource.usecase.PauseDataSource.repo.DetailDataSource: id=%s err=%v", id, err)
		return datasource.PauseOutput{}, datasource.ErrNotFound
	}
	if current.ID == "" {
		return datasource.PauseOutput{}, datasource.ErrNotFound
	}
	if current.Status != model.SourceStatusActive {
		return datasource.PauseOutput{}, datasource.ErrPauseNotAllowed
	}

	updated, err := uc.repo.UpdateDataSource(ctx, repo.UpdateDataSourceOptions{
		ID:     current.ID,
		Status: string(model.SourceStatusPaused),
	})
	if err != nil {
		uc.l.Errorf(ctx, "datasource.usecase.PauseDataSource.repo.UpdateDataSource: id=%s err=%v", id, err)
		return datasource.PauseOutput{}, datasource.ErrUpdateFailed
	}

	return datasource.PauseOutput{DataSource: updated}, nil
}

// ResumeDataSource transitions a PAUSED datasource back to ACTIVE.
func (uc *implUseCase) ResumeDataSource(ctx context.Context, id string) (datasource.ResumeOutput, error) {
	if err := uc.validDataSourceID(id); err != nil {
		uc.l.Warnf(ctx, "datasource.usecase.ResumeDataSource.validDataSourceID: %v", err)
		return datasource.ResumeOutput{}, err
	}

	current, err := uc.repo.DetailDataSource(ctx, strings.TrimSpace(id))
	if err != nil {
		uc.l.Errorf(ctx, "datasource.usecase.ResumeDataSource.repo.DetailDataSource: id=%s err=%v", id, err)
		return datasource.ResumeOutput{}, datasource.ErrNotFound
	}
	if current.ID == "" {
		uc.l.Warnf(ctx, "datasource.usecase.ResumeDataSource.repo.DetailDataSource: id=%s not found", id)
		return datasource.ResumeOutput{}, datasource.ErrNotFound
	}
	if current.Status != model.SourceStatusPaused {
		uc.l.Warnf(ctx, "datasource.usecase.ResumeDataSource: id=%s status=%s not paused", id, current.Status)
		return datasource.ResumeOutput{}, datasource.ErrResumeNotAllowed
	}

	updated, err := uc.repo.UpdateDataSource(ctx, repo.UpdateDataSourceOptions{
		ID:            current.ID,
		Status:        string(model.SourceStatusActive),
		ClearPausedAt: true,
	})
	if err != nil {
		uc.l.Errorf(ctx, "datasource.usecase.ResumeDataSource.repo.UpdateDataSource: id=%s err=%v", id, err)
		return datasource.ResumeOutput{}, datasource.ErrUpdateFailed
	}

	return datasource.ResumeOutput{DataSource: updated}, nil
}

// UpdateCrawlMode updates crawl_mode for an eligible crawl datasource and writes one audit record.
func (uc *implUseCase) UpdateCrawlMode(ctx context.Context, input datasource.UpdateCrawlModeInput) (datasource.UpdateCrawlModeOutput, error) {
	if err := uc.validUpdateCrawlModeInput(input); err != nil {
		uc.l.Warnf(ctx, "datasource.usecase.UpdateCrawlMode.validUpdateCrawlModeInput: %v", err)
		return datasource.UpdateCrawlModeOutput{}, err
	}

	current, err := uc.repo.DetailDataSource(ctx, strings.TrimSpace(input.ID))
	if err != nil {
		uc.l.Errorf(ctx, "datasource.usecase.UpdateCrawlMode.repo.DetailDataSource: id=%s err=%v", input.ID, err)
		return datasource.UpdateCrawlModeOutput{}, datasource.ErrNotFound
	}
	if current.ID == "" {
		uc.l.Warnf(ctx, "datasource.usecase.UpdateCrawlMode.repo.DetailDataSource: id=%s not found", input.ID)
		return datasource.UpdateCrawlModeOutput{}, datasource.ErrNotFound
	}
	if current.SourceCategory != model.SourceCategoryCrawl {
		uc.l.Warnf(ctx, "datasource.usecase.UpdateCrawlMode: id=%s category=%s not a crawl source", input.ID, current.SourceCategory)
		return datasource.UpdateCrawlModeOutput{}, datasource.ErrCrawlModeNotAllowed
	}

	switch current.Status {
	case model.SourceStatusReady, model.SourceStatusActive, model.SourceStatusPaused:
	default:
		uc.l.Warnf(ctx, "datasource.usecase.UpdateCrawlMode: id=%s status=%s not eligible for crawl mode change", input.ID, current.Status)
		return datasource.UpdateCrawlModeOutput{}, datasource.ErrCrawlModeNotAllowed
	}

	if current.CrawlIntervalMinutes == nil || *current.CrawlIntervalMinutes <= 0 || current.CrawlMode == nil {
		uc.l.Warnf(ctx, "datasource.usecase.UpdateCrawlMode: id=%s has invalid crawl configuration", input.ID)
		return datasource.UpdateCrawlModeOutput{}, datasource.ErrCrawlModeNotAllowed
	}

	updated, err := uc.repo.UpdateDataSource(ctx, repo.UpdateDataSourceOptions{
		ID:        current.ID,
		CrawlMode: strings.TrimSpace(input.CrawlMode),
	})
	if err != nil {
		uc.l.Errorf(ctx, "datasource.usecase.UpdateCrawlMode.repo.UpdateDataSource: id=%s err=%v", input.ID, err)
		return datasource.UpdateCrawlModeOutput{}, datasource.ErrUpdateFailed
	}

	triggeredBy, _ := auth.GetUserIDFromContext(ctx)
	if _, err := uc.repo.CreateCrawlModeChange(ctx, repo.CreateCrawlModeChangeOptions{
		SourceID:            current.ID,
		ProjectID:           current.ProjectID,
		TriggerType:         strings.TrimSpace(input.TriggerType),
		FromMode:            string(*current.CrawlMode),
		ToMode:              strings.TrimSpace(input.CrawlMode),
		FromIntervalMinutes: *current.CrawlIntervalMinutes,
		ToIntervalMinutes:   *current.CrawlIntervalMinutes,
		Reason:              strings.TrimSpace(input.Reason),
		EventRef:            strings.TrimSpace(input.EventRef),
		TriggeredBy:         triggeredBy,
	}); err != nil {
		uc.l.Errorf(ctx, "datasource.usecase.UpdateCrawlMode.repo.CreateCrawlModeChange: id=%s err=%v", input.ID, err)
		return datasource.UpdateCrawlModeOutput{}, datasource.ErrUpdateFailed
	}

	return datasource.UpdateCrawlModeOutput{DataSource: updated}, nil
}

// GetActivationReadiness evaluates project-level activation prerequisites from datasource/target data.
func (uc *implUseCase) GetActivationReadiness(ctx context.Context, projectID string) (datasource.ActivationReadinessOutput, error) {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return datasource.ActivationReadinessOutput{}, datasource.ErrProjectIDRequired
	}

	sources, err := uc.repo.ListDataSources(ctx, repo.ListDataSourcesOptions{ProjectID: projectID})
	if err != nil {
		uc.l.Errorf(ctx, "datasource.usecase.GetActivationReadiness.repo.ListDataSources: project_id=%s err=%v", projectID, err)
		return datasource.ActivationReadinessOutput{}, datasource.ErrListFailed
	}

	out := datasource.ActivationReadinessOutput{
		ProjectID:       projectID,
		DataSourceCount: len(sources),
		HasDatasource:   len(sources) > 0,
		Errors:          make([]datasource.ActivationReadinessError, 0),
	}

	if !out.HasDatasource {
		out.Errors = append(out.Errors, datasource.ActivationReadinessError{
			Code:    readinessCodeDatasourceRequired,
			Message: "project must have at least one datasource",
		})
	}

	activeOnly := true
	for _, source := range sources {
		if source.SourceCategory == model.SourceCategoryPassive {
			// TODO: passive confirm flow will have dedicated state transitions. For now,
			// onboarding_status=CONFIRMED is treated as the readiness signal.
			if source.OnboardingStatus != model.OnboardingStatusConfirmed {
				out.PassiveUnconfirmedCount++
				out.Errors = append(out.Errors, datasource.ActivationReadinessError{
					Code:         readinessCodePassiveUnconfirmed,
					Message:      "passive datasource is not confirmed",
					DataSourceID: source.ID,
				})
			}
			continue
		}

		if source.SourceCategory != model.SourceCategoryCrawl {
			continue
		}

		targets, listErr := uc.repo.ListTargets(ctx, repo.ListTargetsOptions{
			DataSourceID: source.ID,
			IsActive:     &activeOnly,
		})
		if listErr != nil {
			uc.l.Errorf(ctx, "datasource.usecase.GetActivationReadiness.repo.ListTargets: source_id=%s err=%v", source.ID, listErr)
			return datasource.ActivationReadinessOutput{}, datasource.ErrListFailed
		}

		for _, target := range targets {
			latest, latestErr := uc.repo.GetLatestDryrunByTarget(ctx, target.ID)
			if latestErr != nil {
				uc.l.Errorf(ctx, "datasource.usecase.GetActivationReadiness.repo.GetLatestDryrunByTarget: target_id=%s err=%v", target.ID, latestErr)
				return datasource.ActivationReadinessOutput{}, datasource.ErrListFailed
			}

			if latest.ID == "" {
				out.MissingTargetDryrunCount++
				out.Errors = append(out.Errors, datasource.ActivationReadinessError{
					Code:         readinessCodeTargetDryrunMiss,
					Message:      "crawl target has never been dry-run",
					DataSourceID: source.ID,
					TargetID:     target.ID,
				})
				continue
			}

			if latest.Status == model.DryrunStatusFailed {
				out.FailedTargetDryrunCount++
				out.Errors = append(out.Errors, datasource.ActivationReadinessError{
					Code:         readinessCodeTargetDryrunFailed,
					Message:      "crawl target latest dry-run is FAILED",
					DataSourceID: source.ID,
					TargetID:     target.ID,
				})
			}
		}
	}

	out.CanActivate = out.HasDatasource &&
		out.PassiveUnconfirmedCount == 0 &&
		out.MissingTargetDryrunCount == 0 &&
		out.FailedTargetDryrunCount == 0

	return out, nil
}

// Activate activates all READY datasources in a project after readiness passes.
func (uc *implUseCase) Activate(ctx context.Context, projectID string) (datasource.ProjectLifecycleOutput, error) {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return datasource.ProjectLifecycleOutput{}, datasource.ErrProjectIDRequired
	}

	readiness, err := uc.GetActivationReadiness(ctx, projectID)
	if err != nil {
		return datasource.ProjectLifecycleOutput{}, err
	}
	if !readiness.CanActivate {
		uc.l.Warnf(ctx, "datasource.usecase.Activate: project_id=%s readiness blocked", projectID)
		return datasource.ProjectLifecycleOutput{}, datasource.ErrActivationReadinessFailed
	}

	sources, err := uc.repo.ListDataSources(ctx, repo.ListDataSourcesOptions{ProjectID: projectID})
	if err != nil {
		uc.l.Errorf(ctx, "datasource.usecase.Activate.repo.ListDataSources: project_id=%s err=%v", projectID, err)
		return datasource.ProjectLifecycleOutput{}, datasource.ErrListFailed
	}

	for _, source := range sources {
		switch source.Status {
		case model.SourceStatusReady, model.SourceStatusActive:
			// allowed
		default:
			uc.l.Warnf(ctx, "datasource.usecase.Activate: project_id=%s source_id=%s status=%s not eligible", projectID, source.ID, source.Status)
			return datasource.ProjectLifecycleOutput{}, datasource.ErrActivateNotAllowed
		}
	}

	affected := 0
	for _, source := range sources {
		if source.Status == model.SourceStatusActive {
			continue
		}

		if _, activateErr := uc.ActivateDataSource(ctx, source.ID); activateErr != nil {
			uc.l.Errorf(ctx, "datasource.usecase.Activate.uc.ActivateDataSource: project_id=%s source_id=%s err=%v", projectID, source.ID, activateErr)
			return datasource.ProjectLifecycleOutput{}, activateErr
		}
		affected++
	}

	return datasource.ProjectLifecycleOutput{
		ProjectID:               projectID,
		AffectedDataSourceCount: affected,
	}, nil
}

// Pause pauses all ACTIVE datasources in a project (idempotent for non-active sources).
func (uc *implUseCase) Pause(ctx context.Context, projectID string) (datasource.ProjectLifecycleOutput, error) {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return datasource.ProjectLifecycleOutput{}, datasource.ErrProjectIDRequired
	}

	sources, err := uc.repo.ListDataSources(ctx, repo.ListDataSourcesOptions{ProjectID: projectID})
	if err != nil {
		uc.l.Errorf(ctx, "datasource.usecase.Pause.repo.ListDataSources: project_id=%s err=%v", projectID, err)
		return datasource.ProjectLifecycleOutput{}, datasource.ErrListFailed
	}

	affected := 0
	for _, source := range sources {
		if source.Status != model.SourceStatusActive {
			continue
		}

		if _, pauseErr := uc.PauseDataSource(ctx, source.ID); pauseErr != nil {
			uc.l.Errorf(ctx, "datasource.usecase.Pause.uc.PauseDataSource: project_id=%s source_id=%s err=%v", projectID, source.ID, pauseErr)
			return datasource.ProjectLifecycleOutput{}, pauseErr
		}
		affected++
	}

	return datasource.ProjectLifecycleOutput{
		ProjectID:               projectID,
		AffectedDataSourceCount: affected,
	}, nil
}

// Resume resumes all PAUSED datasources in a project after readiness passes.
func (uc *implUseCase) Resume(ctx context.Context, projectID string) (datasource.ProjectLifecycleOutput, error) {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return datasource.ProjectLifecycleOutput{}, datasource.ErrProjectIDRequired
	}

	readiness, err := uc.GetActivationReadiness(ctx, projectID)
	if err != nil {
		return datasource.ProjectLifecycleOutput{}, err
	}
	if !readiness.CanActivate {
		uc.l.Warnf(ctx, "datasource.usecase.Resume: project_id=%s readiness blocked", projectID)
		return datasource.ProjectLifecycleOutput{}, datasource.ErrActivationReadinessFailed
	}

	sources, err := uc.repo.ListDataSources(ctx, repo.ListDataSourcesOptions{ProjectID: projectID})
	if err != nil {
		uc.l.Errorf(ctx, "datasource.usecase.Resume.repo.ListDataSources: project_id=%s err=%v", projectID, err)
		return datasource.ProjectLifecycleOutput{}, datasource.ErrListFailed
	}

	for _, source := range sources {
		switch source.Status {
		case model.SourceStatusPaused, model.SourceStatusActive:
			// allowed
		default:
			uc.l.Warnf(ctx, "datasource.usecase.Resume: project_id=%s source_id=%s status=%s not eligible", projectID, source.ID, source.Status)
			return datasource.ProjectLifecycleOutput{}, datasource.ErrResumeNotAllowed
		}
	}

	affected := 0
	for _, source := range sources {
		if source.Status != model.SourceStatusPaused {
			continue
		}

		if _, resumeErr := uc.ResumeDataSource(ctx, source.ID); resumeErr != nil {
			uc.l.Errorf(ctx, "datasource.usecase.Resume.uc.ResumeDataSource: project_id=%s source_id=%s err=%v", projectID, source.ID, resumeErr)
			return datasource.ProjectLifecycleOutput{}, resumeErr
		}
		affected++
	}

	return datasource.ProjectLifecycleOutput{
		ProjectID:               projectID,
		AffectedDataSourceCount: affected,
	}, nil
}
