package usecase

import (
	"context"
	"strings"

	"ingest-srv/internal/execution"
	repo "ingest-srv/internal/execution/repository"
)

func (uc *implUseCase) CancelProjectRuntime(ctx context.Context, input execution.CancelProjectRuntimeInput) error {
	projectID := strings.TrimSpace(input.ProjectID)
	if projectID == "" {
		return execution.ErrCancelRuntimeFailed
	}

	if err := uc.repo.CancelProjectRuntime(ctx, repo.CancelProjectRuntimeOptions{
		ProjectID:  projectID,
		Reason:     strings.TrimSpace(input.Reason),
		CanceledAt: input.CanceledAt,
	}); err != nil {
		uc.l.Errorf(ctx, "execution.usecase.CancelProjectRuntime.repo.CancelProjectRuntime: project_id=%s err=%v", projectID, err)
		return execution.ErrCancelRuntimeFailed
	}

	return nil
}
