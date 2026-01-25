package usecase

import (
	"context"

	"smap-collector/internal/webhook"
	"smap-collector/pkg/project"
)

// NotifyProgress gửi progress update tới Project Service.
// Two-phase format: crawl + analyze progress
func (uc *implUseCase) NotifyProgress(ctx context.Context, req webhook.ProgressRequest) error {
	if !req.IsValid() {
		return webhook.ErrInvalidRequest
	}

	// Convert to pkg/project request with two-phase format
	projectReq := project.ProgressCallbackRequest{
		ProjectID: req.ProjectID,
		UserID:    req.UserID,
		Status:    req.Status,
		Crawl: project.PhaseProgressCallback{
			Total:           req.Crawl.Total,
			Done:            req.Crawl.Done,
			Errors:          req.Crawl.Errors,
			ProgressPercent: req.Crawl.ProgressPercent,
		},
		Analyze: project.PhaseProgressCallback{
			Total:           req.Analyze.Total,
			Done:            req.Analyze.Done,
			Errors:          req.Analyze.Errors,
			ProgressPercent: req.Analyze.ProgressPercent,
		},
		OverallProgressPercent: req.OverallProgressPercent,
	}

	if err := uc.projectClient.SendProgressCallback(ctx, projectReq); err != nil {
		uc.l.Warnf(ctx, "NotifyProgress failed: project_id=%s, error=%v", req.ProjectID, err)
		return err
	}

	uc.l.Infof(ctx, "Progress notified: project_id=%s, status=%s, overall_progress=%.1f%%",
		req.ProjectID, req.Status, req.OverallProgressPercent)
	return nil
}

// NotifyCompletion gửi completion notification.
// Two-phase format: crawl + analyze progress
func (uc *implUseCase) NotifyCompletion(ctx context.Context, req webhook.ProgressRequest) error {
	if !req.IsValid() {
		return webhook.ErrInvalidRequest
	}

	// Convert to pkg/project request with two-phase format
	projectReq := project.ProgressCallbackRequest{
		ProjectID: req.ProjectID,
		UserID:    req.UserID,
		Status:    req.Status,
		Crawl: project.PhaseProgressCallback{
			Total:           req.Crawl.Total,
			Done:            req.Crawl.Done,
			Errors:          req.Crawl.Errors,
			ProgressPercent: req.Crawl.ProgressPercent,
		},
		Analyze: project.PhaseProgressCallback{
			Total:           req.Analyze.Total,
			Done:            req.Analyze.Done,
			Errors:          req.Analyze.Errors,
			ProgressPercent: req.Analyze.ProgressPercent,
		},
		OverallProgressPercent: req.OverallProgressPercent,
	}

	if err := uc.projectClient.SendProgressCallback(ctx, projectReq); err != nil {
		uc.l.Errorf(ctx, "NotifyCompletion failed: project_id=%s, error=%v", req.ProjectID, err)
		return err
	}

	uc.l.Infof(ctx, "Completion notified: project_id=%s, status=%s", req.ProjectID, req.Status)
	return nil
}
