package usecase

import (
	"context"

	"smap-collector/internal/models"
	"smap-collector/internal/webhook"
)

// HandleProjectCreatedEvent xử lý ProjectCreatedEvent từ Project Service.
func (uc implUseCase) HandleProjectCreatedEvent(ctx context.Context, event models.ProjectCreatedEvent) error {
	// Validate event
	if !event.IsValid() {
		uc.l.Warnf(ctx, "dispatcher.usecase.project_event.HandleProjectCreatedEvent: invalid event - event_id=%s", event.EventID)
		return ErrInvalidProjectEvent
	}

	projectID := event.Payload.ProjectID
	userID := event.Payload.UserID

	uc.l.Infof(ctx, "dispatcher.usecase.project_event.HandleProjectCreatedEvent: processing event - event_id=%s, project_id=%s, user_id=%s",
		event.EventID, projectID, userID)

	// Store user mapping for progress notifications (if state usecase is available)
	if uc.stateUC != nil {
		if err := uc.stateUC.StoreUserMapping(ctx, projectID, userID); err != nil {
			uc.l.Warnf(ctx, "dispatcher.usecase.project_event.HandleProjectCreatedEvent: failed to store user mapping: %v", err)
			// Continue processing even if mapping fails
		}
	}

	// Transform event to CrawlRequests
	requests := models.TransformProjectEventToRequests(event, models.DefaultTransformOptions())
	totalTasks := len(requests) * len(uc.selectPlatforms()) // Each request goes to all platforms

	uc.l.Infof(ctx, "dispatcher.usecase.project_event.HandleProjectCreatedEvent: transformed to %d requests, total tasks=%d",
		len(requests), totalTasks)

	// Set crawl total in Redis state and notify (if state usecase is available)
	if uc.stateUC != nil {
		if err := uc.stateUC.SetCrawlTotal(ctx, projectID, int64(totalTasks)); err != nil {
			uc.l.Warnf(ctx, "dispatcher.usecase.project_event.HandleProjectCreatedEvent: failed to set crawl total: %v", err)
		}

		// Notify progress after setting total
		if uc.webhookUC != nil {
			uc.l.Infof(ctx, "dispatcher.usecase.project_event.HandleProjectCreatedEvent: notifying progress - project_id=%s, user_id=%s", projectID, userID)
			uc.notifyProgress(ctx, projectID, userID)
		}
	}

	// Dispatch each request to workers
	successCount := 0
	errorCount := 0

	for _, req := range requests {
		tasks, err := uc.Dispatch(ctx, req)
		if err != nil {
			uc.l.Errorf(ctx, "dispatcher.usecase.project_event.HandleProjectCreatedEvent: failed to dispatch job_id=%s: %v", req.JobID, err)
			platformCount := int64(len(uc.selectPlatforms()))
			errorCount += int(platformCount)

			// Update crawl error count in state
			if uc.stateUC != nil {
				_ = uc.stateUC.IncrementCrawlErrorsBy(ctx, projectID, platformCount)
			}
			continue
		}

		successCount += len(tasks)
	}

	return nil
}

// notifyProgress sends progress notification.
func (uc implUseCase) notifyProgress(ctx context.Context, projectID, userID string) {
	if uc.webhookUC == nil || uc.stateUC == nil {
		return
	}

	state, err := uc.stateUC.GetState(ctx, projectID)
	if err != nil {
		uc.l.Warnf(ctx, "dispatcher.usecase.project_event.notifyProgress: failed to get state: %v", err)
		return
	}

	req := uc.buildProgressRequest(projectID, userID, state)
	if err := uc.webhookUC.NotifyProgress(ctx, req); err != nil {
		uc.l.Warnf(ctx, "dispatcher.usecase.project_event.notifyProgress: failed to notify: %v", err)
	}
}

// buildProgressRequest builds a webhook progress request from state with two-phase format.
func (uc implUseCase) buildProgressRequest(projectID, userID string, state *models.ProjectState) webhook.ProgressRequest {
	return webhook.ProgressRequest{
		ProjectID: projectID,
		UserID:    userID,
		Status:    string(state.Status),
		Crawl: webhook.PhaseProgress{
			Total:           state.CrawlTotal,
			Done:            state.CrawlDone,
			Errors:          state.CrawlErrors,
			ProgressPercent: state.CrawlProgressPercent(),
		},
		Analyze: webhook.PhaseProgress{
			Total:           state.AnalyzeTotal,
			Done:            state.AnalyzeDone,
			Errors:          state.AnalyzeErrors,
			ProgressPercent: state.AnalyzeProgressPercent(),
		},
		OverallProgressPercent: state.OverallProgressPercent(),
	}
}
