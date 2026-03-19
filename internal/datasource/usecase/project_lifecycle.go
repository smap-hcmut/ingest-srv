package usecase

import (
	"context"
	"strings"

	"ingest-srv/internal/datasource"
	repo "ingest-srv/internal/datasource/repository"
	"ingest-srv/internal/model"
)

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
			Code:    datasource.ActivationReadinessCodeDatasourceRequired,
			Message: "project must have at least one datasource",
		})
	}

	for _, source := range sources {
		if source.SourceCategory == model.SourceCategoryPassive {
			// TODO: passive confirm flow will have dedicated state transitions. For now,
			// onboarding_status=CONFIRMED is treated as the readiness signal.
			if source.OnboardingStatus != model.OnboardingStatusConfirmed {
				out.PassiveUnconfirmedCount++
				out.Errors = append(out.Errors, datasource.ActivationReadinessError{
					Code:         datasource.ActivationReadinessCodePassiveUnconfirmed,
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
					Code:         datasource.ActivationReadinessCodeTargetDryrunMiss,
					Message:      "crawl target has never been dry-run",
					DataSourceID: source.ID,
					TargetID:     target.ID,
				})
				continue
			}

			if latest.Status == model.DryrunStatusFailed {
				out.FailedTargetDryrunCount++
				out.Errors = append(out.Errors, datasource.ActivationReadinessError{
					Code:         datasource.ActivationReadinessCodeTargetDryrunFailed,
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
