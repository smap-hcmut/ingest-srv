package usecase

import (
	"context"
	"strings"

	"ingest-srv/internal/datasource"
	repo "ingest-srv/internal/datasource/repository"
	"ingest-srv/internal/model"

	"github.com/smap-hcmut/shared-libs/go/auth"
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

	if err := uc.ensureRuntimePrerequisites(ctx, current, datasource.ErrActivateNotAllowed); err != nil {
		return datasource.ActivateOutput{}, err
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
	if err := uc.ensureRuntimePrerequisites(ctx, current, datasource.ErrResumeNotAllowed); err != nil {
		return datasource.ResumeOutput{}, err
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
