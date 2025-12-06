package usecase

import (
	"context"

	"smap-collector/internal/webhook"
	"smap-collector/pkg/project"
)

// NotifyProgress gửi progress update tới Project Service.
func (uc *implUseCase) NotifyProgress(ctx context.Context, req webhook.ProgressRequest) error {
	if !req.IsValid() {
		return webhook.ErrInvalidRequest
	}

	// Convert to pkg/project request
	projectReq := project.ProgressCallbackRequest{
		ProjectID: req.ProjectID,
		UserID:    req.UserID,
		Status:    req.Status,
		Total:     req.Total,
		Done:      req.Done,
		Errors:    req.Errors,
	}

	if err := uc.projectClient.SendProgressCallback(ctx, projectReq); err != nil {
		uc.l.Warnf(ctx, "NotifyProgress failed: project_id=%s, error=%v", req.ProjectID, err)
		return err
	}

	uc.l.Infof(ctx, "Progress notified: project_id=%s, status=%s, progress=%.1f%%",
		req.ProjectID, req.Status, req.ProgressPercent())
	return nil
}

// NotifyCompletion gửi completion notification.
func (uc *implUseCase) NotifyCompletion(ctx context.Context, req webhook.ProgressRequest) error {
	if !req.IsValid() {
		return webhook.ErrInvalidRequest
	}

	// Convert to pkg/project request
	projectReq := project.ProgressCallbackRequest{
		ProjectID: req.ProjectID,
		UserID:    req.UserID,
		Status:    req.Status,
		Total:     req.Total,
		Done:      req.Done,
		Errors:    req.Errors,
	}

	if err := uc.projectClient.SendProgressCallback(ctx, projectReq); err != nil {
		uc.l.Errorf(ctx, "NotifyCompletion failed: project_id=%s, error=%v", req.ProjectID, err)
		return err
	}

	uc.l.Infof(ctx, "Completion notified: project_id=%s, status=%s", req.ProjectID, req.Status)
	return nil
}
