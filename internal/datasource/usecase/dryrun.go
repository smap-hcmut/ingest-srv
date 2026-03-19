package usecase

import (
	"context"
	"strings"

	"ingest-srv/internal/datasource"
	repo "ingest-srv/internal/datasource/repository"
	"ingest-srv/internal/model"
)

// MarkDryrunRunning updates datasource dryrun state at dry-run dispatch time.
func (uc *implUseCase) MarkDryrunRunning(ctx context.Context, input datasource.MarkDryrunRunningInput) (datasource.MarkDryrunRunningOutput, error) {
	current, err := uc.repo.DetailDataSource(ctx, strings.TrimSpace(input.ID))
	if err != nil {
		uc.l.Errorf(ctx, "datasource.usecase.MarkDryrunRunning.repo.DetailDataSource: id=%s err=%v", input.ID, err)
		return datasource.MarkDryrunRunningOutput{}, datasource.ErrUpdateFailed
	}
	if current.ID == "" {
		return datasource.MarkDryrunRunningOutput{}, datasource.ErrNotFound
	}

	opt := repo.UpdateDataSourceOptions{
		ID:                 current.ID,
		DryrunStatus:       string(model.DryrunStatusRunning),
		DryrunLastResultID: strings.TrimSpace(input.DryrunLastResultID),
	}
	if current.Status == model.SourceStatusPending || current.Status == model.SourceStatusReady {
		opt.Status = string(model.SourceStatusPending)
	}

	updated, err := uc.repo.UpdateDataSource(ctx, opt)
	if err != nil {
		uc.l.Errorf(ctx, "datasource.usecase.MarkDryrunRunning.repo.UpdateDataSource: id=%s err=%v", input.ID, err)
		return datasource.MarkDryrunRunningOutput{}, datasource.ErrUpdateFailed
	}

	return datasource.MarkDryrunRunningOutput{DataSource: updated}, nil
}

// ApplyDryrunResult updates datasource dryrun state when one dry-run result is finalized.
func (uc *implUseCase) ApplyDryrunResult(ctx context.Context, input datasource.ApplyDryrunResultInput) (datasource.ApplyDryrunResultOutput, error) {
	current, err := uc.repo.DetailDataSource(ctx, strings.TrimSpace(input.ID))
	if err != nil {
		uc.l.Errorf(ctx, "datasource.usecase.ApplyDryrunResult.repo.DetailDataSource: id=%s err=%v", input.ID, err)
		return datasource.ApplyDryrunResultOutput{}, datasource.ErrUpdateFailed
	}
	if current.ID == "" {
		return datasource.ApplyDryrunResultOutput{}, datasource.ErrNotFound
	}

	opt := repo.UpdateDataSourceOptions{
		ID:                 current.ID,
		DryrunStatus:       strings.TrimSpace(input.DryrunStatus),
		DryrunLastResultID: strings.TrimSpace(input.DryrunLastResultID),
	}

	if current.Status == model.SourceStatusPending || current.Status == model.SourceStatusReady {
		if model.DryrunStatus(strings.TrimSpace(input.DryrunStatus)) == model.DryrunStatusFailed {
			opt.Status = string(model.SourceStatusPending)
		} else {
			opt.Status = string(model.SourceStatusReady)
		}
	}

	updated, err := uc.repo.UpdateDataSource(ctx, opt)
	if err != nil {
		uc.l.Errorf(ctx, "datasource.usecase.ApplyDryrunResult.repo.UpdateDataSource: id=%s err=%v", input.ID, err)
		return datasource.ApplyDryrunResultOutput{}, datasource.ErrUpdateFailed
	}

	return datasource.ApplyDryrunResultOutput{DataSource: updated}, nil
}
