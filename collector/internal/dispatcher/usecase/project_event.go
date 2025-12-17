package usecase

import (
	"context"

	"smap-collector/internal/models"
	"smap-collector/internal/webhook"
)

// HandleProjectCreatedEvent xử lý ProjectCreatedEvent từ Project Service.
// Sử dụng config-driven limits thay vì hardcode values.
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

	// Transform event to CrawlRequests using config-driven limits
	opts := models.NewTransformOptionsFromConfig(uc.crawlLimitsCfg)
	requests := models.TransformProjectEventToRequests(event, opts)
	totalTasks := int64(len(requests) * len(uc.selectPlatforms())) // Each request goes to all platforms

	// Calculate items expected (for item-level progress tracking)
	itemsExpected := totalTasks * int64(opts.LimitPerKeyword)

	uc.l.Infof(ctx, "dispatcher.usecase.project_event.HandleProjectCreatedEvent: transformed to %d requests, total_tasks=%d, items_expected=%d, limit_per_keyword=%d",
		len(requests), totalTasks, itemsExpected, opts.LimitPerKeyword)

	// Set tasks total and items expected in Redis state (hybrid state tracking)
	if uc.stateUC != nil {
		if err := uc.stateUC.SetTasksTotal(ctx, projectID, totalTasks, itemsExpected); err != nil {
			uc.l.Warnf(ctx, "dispatcher.usecase.project_event.HandleProjectCreatedEvent: failed to set tasks total: %v", err)
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
			platformCount := len(uc.selectPlatforms())
			errorCount += platformCount

			// Update task errors in state (1 error per platform)
			if uc.stateUC != nil {
				for i := 0; i < platformCount; i++ {
					_ = uc.stateUC.IncrementTasksErrors(ctx, projectID)
				}
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

// buildProgressRequest builds a webhook progress request from state with hybrid format.
// Includes both task-level and item-level progress for crawl phase.
func (uc implUseCase) buildProgressRequest(projectID, userID string, state *models.ProjectState) webhook.ProgressRequest {
	return webhook.ProgressRequest{
		ProjectID: projectID,
		UserID:    userID,
		Status:    string(state.Status),
		// Task-level progress (for completion tracking)
		Tasks: webhook.TaskProgress{
			Total:   state.TasksTotal,
			Done:    state.TasksDone,
			Errors:  state.TasksErrors,
			Percent: state.TasksProgressPercent(),
		},
		// Item-level progress (for progress display)
		Items: webhook.ItemProgress{
			Expected: state.ItemsExpected,
			Actual:   state.ItemsActual,
			Errors:   state.ItemsErrors,
			Percent:  state.ItemsProgressPercent(),
		},
		// Legacy crawl progress (for backward compatibility)
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
