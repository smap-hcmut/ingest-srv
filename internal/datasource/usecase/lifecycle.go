package usecase

import (
	"context"
	"strings"

	"ingest-srv/internal/datasource"
	repo "ingest-srv/internal/datasource/repository"
	"ingest-srv/internal/model"
	"ingest-srv/pkg/scope"
)

// Activate transitions a datasource from READY to ACTIVE when runtime prerequisites are met.
func (uc *implUseCase) Activate(ctx context.Context, input datasource.ActivateInput) (datasource.ActivateOutput, error) {
	if err := uc.validActivateInput(input); err != nil {
		uc.l.Warnf(ctx, "datasource.usecase.Activate.validActivateInput: %v", err)
		return datasource.ActivateOutput{}, err
	}

	current, err := uc.repo.DetailDataSource(ctx, strings.TrimSpace(input.ID))
	if err != nil {
		uc.l.Errorf(ctx, "datasource.usecase.Activate.repo.DetailDataSource: id=%s err=%v", input.ID, err)
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
			uc.l.Errorf(ctx, "datasource.usecase.Activate.repo.CountActiveTargets: id=%s err=%v", input.ID, err)
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
		uc.l.Errorf(ctx, "datasource.usecase.Activate.repo.UpdateDataSource: id=%s err=%v", input.ID, err)
		return datasource.ActivateOutput{}, datasource.ErrUpdateFailed
	}

	return datasource.ActivateOutput{DataSource: updated}, nil
}

// Pause transitions an ACTIVE datasource into PAUSED.
func (uc *implUseCase) Pause(ctx context.Context, input datasource.PauseInput) (datasource.PauseOutput, error) {
	if err := uc.validPauseInput(input); err != nil {
		uc.l.Warnf(ctx, "datasource.usecase.Pause.validPauseInput: %v", err)
		return datasource.PauseOutput{}, err
	}

	current, err := uc.repo.DetailDataSource(ctx, strings.TrimSpace(input.ID))
	if err != nil {
		uc.l.Errorf(ctx, "datasource.usecase.Pause.repo.DetailDataSource: id=%s err=%v", input.ID, err)
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
		uc.l.Errorf(ctx, "datasource.usecase.Pause.repo.UpdateDataSource: id=%s err=%v", input.ID, err)
		return datasource.PauseOutput{}, datasource.ErrUpdateFailed
	}

	return datasource.PauseOutput{DataSource: updated}, nil
}

// Resume transitions a PAUSED datasource back to ACTIVE.
func (uc *implUseCase) Resume(ctx context.Context, input datasource.ResumeInput) (datasource.ResumeOutput, error) {
	if err := uc.validResumeInput(input); err != nil {
		uc.l.Warnf(ctx, "datasource.usecase.Resume.validResumeInput: %v", err)
		return datasource.ResumeOutput{}, err
	}

	current, err := uc.repo.DetailDataSource(ctx, strings.TrimSpace(input.ID))
	if err != nil {
		uc.l.Errorf(ctx, "datasource.usecase.Resume.repo.DetailDataSource: id=%s err=%v", input.ID, err)
		return datasource.ResumeOutput{}, datasource.ErrNotFound
	}
	if current.ID == "" {
		uc.l.Warnf(ctx, "datasource.usecase.Resume.repo.DetailDataSource: id=%s not found", input.ID)
		return datasource.ResumeOutput{}, datasource.ErrNotFound
	}
	if current.Status != model.SourceStatusPaused {
		uc.l.Warnf(ctx, "datasource.usecase.Resume: id=%s status=%s not paused", input.ID, current.Status)
		return datasource.ResumeOutput{}, datasource.ErrResumeNotAllowed
	}

	updated, err := uc.repo.UpdateDataSource(ctx, repo.UpdateDataSourceOptions{
		ID:            current.ID,
		Status:        string(model.SourceStatusActive),
		ClearPausedAt: true,
	})
	if err != nil {
		uc.l.Errorf(ctx, "datasource.usecase.Resume.repo.UpdateDataSource: id=%s err=%v", input.ID, err)
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

	triggeredBy, _ := scope.GetUserIDFromContext(ctx)
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
