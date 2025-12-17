package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"smap-collector/internal/models"
	"smap-collector/internal/results"
	"smap-collector/internal/webhook"
	"smap-collector/pkg/project"
)

// TaskTypeAnalyzeResult is the task type for analyze results from Analytics Service
const TaskTypeAnalyzeResult = "analyze_result"

func (uc implUseCase) HandleResult(ctx context.Context, res models.CrawlerResult) error {
	uc.l.Infof(ctx, "results.usecase.result.HandleResult: handling crawler result: success=%v", res.Success)

	// Extract task_type from result to determine handling strategy
	taskType := uc.extractTaskType(ctx, res.Payload)

	// Route to appropriate handler based on task_type
	switch taskType {
	case string(models.TaskTypeDryRunKeyword):
		return uc.handleDryRunResult(ctx, res)
	case string(models.TaskTypeResearchAndCrawl):
		return uc.handleProjectResult(ctx, res)
	case TaskTypeAnalyzeResult:
		return uc.handleAnalyzeResult(ctx, res)
	default:
		// Default to dry-run for backward compatibility
		uc.l.Warnf(ctx, "results.usecase.result.HandleResult: unknown task_type '%s', defaulting to dry-run handler", taskType)
		return uc.handleDryRunResult(ctx, res)
	}
}

// extractTaskType extracts task_type from crawler result payload
// Supports both crawler content array format and analyze result format
func (uc implUseCase) extractTaskType(ctx context.Context, payload any) string {
	if payload == nil {
		uc.l.Warnf(ctx, "results.usecase.result.extractTaskType: payload is nil")
		return ""
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		uc.l.Warnf(ctx, "results.usecase.result.extractTaskType: failed to marshal payload: %v", err)
		return ""
	}

	// Try to parse as AnalyzeResultPayload first (single object with task_type field)
	var analyzePayload results.AnalyzeResultPayload
	if err := json.Unmarshal(jsonData, &analyzePayload); err == nil && analyzePayload.TaskType == TaskTypeAnalyzeResult {
		uc.l.Infof(ctx, "results.usecase.result.extractTaskType: extracted task_type from analyze payload: %s", analyzePayload.TaskType)
		return analyzePayload.TaskType
	}

	// Try to parse as crawler content array
	var crawlerContentArray []results.CrawlerContent
	if err := json.Unmarshal(jsonData, &crawlerContentArray); err != nil {
		uc.l.Warnf(ctx, "results.usecase.result.extractTaskType: failed to unmarshal payload: %v", err)
		return ""
	}

	if len(crawlerContentArray) > 0 {
		uc.l.Infof(ctx, "results.usecase.result.extractTaskType: extracted task_type: %s", crawlerContentArray[0].Meta.TaskType)
		return crawlerContentArray[0].Meta.TaskType
	}

	uc.l.Warnf(ctx, "results.usecase.result.extractTaskType: no task_type found")
	return ""
}

// handleDryRunResult handles dry-run results by sending callback to Project Service
func (uc implUseCase) handleDryRunResult(ctx context.Context, res models.CrawlerResult) error {
	uc.l.Infof(ctx, "results.usecase.result.handleDryRunResult: handling dry-run result")

	// Build callback request
	callbackReq, err := uc.buildCallbackRequest(ctx, res)
	if err != nil {
		uc.l.Errorf(ctx, "results.usecase.result.handleDryRunResult: failed to build callback request: success=%v, error=%v",
			res.Success, err)
		return fmt.Errorf("%w: %v", results.ErrInvalidInput, err)
	}

	// Send webhook to project service (dry-run callback)
	if err := uc.projectClient.SendDryRunCallback(ctx, callbackReq); err != nil {
		return uc.handleWebhookError(ctx, callbackReq.JobID, callbackReq.Platform, err)
	}

	uc.l.Infof(ctx, "results.usecase.result.handleDryRunResult: successfully sent dry-run callback for job_id=%s, platform=%s",
		callbackReq.JobID, callbackReq.Platform)

	return nil
}

// handleProjectResult handles project execution results by updating state and sending progress webhook.
// Hybrid state: updates task-level (completion) and item-level (progress) counters.
// Supports enhanced Crawler response with limit_info and stats, with fallback to old format.
func (uc implUseCase) handleProjectResult(ctx context.Context, res models.CrawlerResult) error {
	uc.l.Infof(ctx, "results.usecase.result.handleProjectResult: handling project execution result")

	// Extract project_id from job_id (format: {projectID}-brand-{index})
	projectID, err := uc.extractProjectID(ctx, res.Payload)
	if err != nil {
		uc.l.Errorf(ctx, "results.usecase.result.handleProjectResult: failed to extract project_id: %v", err)
		return fmt.Errorf("%w: %v", results.ErrInvalidInput, err)
	}

	// Extract limit_info and stats from enhanced response (with fallback)
	limitInfo, stats := uc.extractLimitInfoAndStats(ctx, res)

	// =========================================================================
	// STEP 1: Update task-level counter (always 1 per response)
	// =========================================================================
	if res.Success {
		if err := uc.stateUC.IncrementTasksDone(ctx, projectID); err != nil {
			uc.l.Errorf(ctx, "results.usecase.result.handleProjectResult: failed to increment tasks_done: project_id=%s, error=%v", projectID, err)
			return fmt.Errorf("%w: %v", results.ErrTemporary, err)
		}
	} else {
		if err := uc.stateUC.IncrementTasksErrors(ctx, projectID); err != nil {
			uc.l.Errorf(ctx, "results.usecase.result.handleProjectResult: failed to increment tasks_errors: project_id=%s, error=%v", projectID, err)
			return fmt.Errorf("%w: %v", results.ErrTemporary, err)
		}
	}

	// =========================================================================
	// STEP 2: Update item-level counters (from stats)
	// =========================================================================
	if stats.Successful > 0 {
		if err := uc.stateUC.IncrementItemsActualBy(ctx, projectID, int64(stats.Successful)); err != nil {
			uc.l.Errorf(ctx, "results.usecase.result.handleProjectResult: failed to increment items_actual: project_id=%s, count=%d, error=%v", projectID, stats.Successful, err)
			return fmt.Errorf("%w: %v", results.ErrTemporary, err)
		}
		// Auto-increment analyze_total (each successful item = 1 item to analyze)
		if err := uc.stateUC.IncrementAnalyzeTotalBy(ctx, projectID, int64(stats.Successful)); err != nil {
			uc.l.Errorf(ctx, "results.usecase.result.handleProjectResult: failed to increment analyze_total: project_id=%s, count=%d, error=%v", projectID, stats.Successful, err)
			return fmt.Errorf("%w: %v", results.ErrTemporary, err)
		}
	}
	if stats.Failed > 0 {
		if err := uc.stateUC.IncrementItemsErrorsBy(ctx, projectID, int64(stats.Failed)); err != nil {
			uc.l.Errorf(ctx, "results.usecase.result.handleProjectResult: failed to increment items_errors: project_id=%s, count=%d, error=%v", projectID, stats.Failed, err)
			return fmt.Errorf("%w: %v", results.ErrTemporary, err)
		}
	}

	// =========================================================================
	// STEP 3: Log platform limitation warning
	// =========================================================================
	if limitInfo != nil && limitInfo.PlatformLimited {
		uc.l.Warnf(ctx, "results.usecase.result.handleProjectResult: platform limited - project_id=%s, requested=%d, found=%d",
			projectID, limitInfo.RequestedLimit, limitInfo.TotalFound)
	}

	// =========================================================================
	// STEP 4: Get current state for progress notification
	// =========================================================================
	state, err := uc.stateUC.GetState(ctx, projectID)
	if err != nil {
		uc.l.Errorf(ctx, "results.usecase.result.handleProjectResult: failed to get state: project_id=%s, error=%v", projectID, err)
		return fmt.Errorf("%w: %v", results.ErrTemporary, err)
	}

	// Get user_id for webhook
	userID, err := uc.stateUC.GetUserID(ctx, projectID)
	if err != nil {
		uc.l.Warnf(ctx, "results.usecase.result.handleProjectResult: failed to get user_id, using empty: project_id=%s, error=%v", projectID, err)
		userID = ""
	}

	// =========================================================================
	// STEP 5: Send progress webhook with hybrid format
	// =========================================================================
	progressReq := uc.buildHybridProgressRequest(projectID, userID, state)

	if err := uc.webhookUC.NotifyProgress(ctx, progressReq); err != nil {
		uc.l.Warnf(ctx, "results.usecase.result.handleProjectResult: failed to send progress webhook (non-fatal): project_id=%s, error=%v", projectID, err)
		// Don't return error - progress webhook failure is non-fatal
	}

	// =========================================================================
	// STEP 6: Check if project is complete (both crawl and analyze phases)
	// =========================================================================
	completed, err := uc.stateUC.CheckCompletion(ctx, projectID)
	if err != nil {
		uc.l.Warnf(ctx, "results.usecase.result.handleProjectResult: failed to check completion: project_id=%s, error=%v", projectID, err)
	} else if completed {
		uc.l.Infof(ctx, "results.usecase.result.handleProjectResult: project completed: project_id=%s", projectID)
		// Send completion notification
		if err := uc.webhookUC.NotifyCompletion(ctx, progressReq); err != nil {
			uc.l.Warnf(ctx, "results.usecase.result.handleProjectResult: failed to send completion webhook: project_id=%s, error=%v", projectID, err)
		}
	}

	return nil
}

// extractLimitInfoAndStats extracts limit_info and stats from enhanced Crawler response.
// Falls back to counting payload items if enhanced fields are not present.
func (uc implUseCase) extractLimitInfoAndStats(ctx context.Context, res models.CrawlerResult) (*models.LimitInfo, *models.CrawlStats) {
	// Try to parse as EnhancedCrawlerResult
	jsonData, err := json.Marshal(res)
	if err != nil {
		return uc.fallbackLimitInfoAndStats(ctx, res)
	}

	var enhanced models.EnhancedCrawlerResult
	if err := json.Unmarshal(jsonData, &enhanced); err != nil {
		return uc.fallbackLimitInfoAndStats(ctx, res)
	}

	// Check if enhanced fields exist
	if enhanced.LimitInfo != nil && enhanced.Stats != nil {
		uc.l.Debugf(ctx, "results.usecase.result.extractLimitInfoAndStats: using enhanced response format")
		return enhanced.LimitInfo, enhanced.Stats
	}

	// Fallback to old format
	return uc.fallbackLimitInfoAndStats(ctx, res)
}

// fallbackLimitInfoAndStats creates limit_info and stats from old response format by counting payload items.
func (uc implUseCase) fallbackLimitInfoAndStats(ctx context.Context, res models.CrawlerResult) (*models.LimitInfo, *models.CrawlStats) {
	itemCount := uc.countBatchItems(ctx, res.Payload)
	if itemCount == 0 {
		itemCount = 1 // Default to 1 if can't determine batch size
	}

	uc.l.Debugf(ctx, "results.usecase.result.fallbackLimitInfoAndStats: using fallback (counted %d items)", itemCount)

	// For fallback, assume all items are successful if res.Success is true
	successful := 0
	failed := 0
	if res.Success {
		successful = itemCount
	} else {
		failed = itemCount
	}

	return &models.LimitInfo{
			RequestedLimit:  50, // default
			AppliedLimit:    50,
			TotalFound:      itemCount,
			PlatformLimited: false,
		}, &models.CrawlStats{
			Successful:     successful,
			Failed:         failed,
			Skipped:        0,
			CompletionRate: 1.0,
		}
}

// buildHybridProgressRequest builds a webhook.ProgressRequest with hybrid state (task-level + item-level).
func (uc implUseCase) buildHybridProgressRequest(projectID, userID string, state *models.ProjectState) webhook.ProgressRequest {
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

// countBatchItems counts the number of items in a crawler result payload
func (uc implUseCase) countBatchItems(ctx context.Context, payload any) int {
	if payload == nil {
		return 0
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		uc.l.Warnf(ctx, "results.usecase.result.countBatchItems: failed to marshal payload: %v", err)
		return 0
	}

	var crawlerContentArray []results.CrawlerContent
	if err := json.Unmarshal(jsonData, &crawlerContentArray); err != nil {
		uc.l.Warnf(ctx, "results.usecase.result.countBatchItems: failed to unmarshal payload: %v", err)
		return 0
	}

	return len(crawlerContentArray)
}

// buildTwoPhaseProgressRequest builds a webhook.ProgressRequest with two-phase state.
// Deprecated: Use buildHybridProgressRequest instead for hybrid state tracking.
func (uc implUseCase) buildTwoPhaseProgressRequest(projectID, userID string, state *models.ProjectState) webhook.ProgressRequest {
	// Delegate to hybrid progress request for consistency
	return uc.buildHybridProgressRequest(projectID, userID, state)
}

// handleAnalyzeResult handles analyze results from Analytics Service
// Updates analyze counters and checks for project completion
func (uc implUseCase) handleAnalyzeResult(ctx context.Context, res models.CrawlerResult) error {
	uc.l.Infof(ctx, "results.usecase.result.handleAnalyzeResult: handling analyze result")

	// Extract analyze payload
	payload, err := uc.extractAnalyzePayload(ctx, res.Payload)
	if err != nil {
		uc.l.Errorf(ctx, "results.usecase.result.handleAnalyzeResult: failed to extract analyze payload: %v", err)
		return fmt.Errorf("%w: %v", results.ErrInvalidInput, err)
	}

	projectID := payload.ProjectID
	if projectID == "" {
		uc.l.Errorf(ctx, "results.usecase.result.handleAnalyzeResult: project_id is empty")
		return results.ErrInvalidInput
	}

	uc.l.Infof(ctx, "results.usecase.result.handleAnalyzeResult: processing analyze result: project_id=%s, job_id=%s, success=%d, errors=%d",
		projectID, payload.JobID, payload.SuccessCount, payload.ErrorCount)

	// Update analyze counters
	if payload.SuccessCount > 0 {
		if err := uc.stateUC.IncrementAnalyzeDoneBy(ctx, projectID, int64(payload.SuccessCount)); err != nil {
			uc.l.Errorf(ctx, "results.usecase.result.handleAnalyzeResult: failed to increment analyze_done: project_id=%s, count=%d, error=%v",
				projectID, payload.SuccessCount, err)
			return fmt.Errorf("%w: %v", results.ErrTemporary, err)
		}
	}

	if payload.ErrorCount > 0 {
		if err := uc.stateUC.IncrementAnalyzeErrorsBy(ctx, projectID, int64(payload.ErrorCount)); err != nil {
			uc.l.Errorf(ctx, "results.usecase.result.handleAnalyzeResult: failed to increment analyze_errors: project_id=%s, count=%d, error=%v",
				projectID, payload.ErrorCount, err)
			return fmt.Errorf("%w: %v", results.ErrTemporary, err)
		}
	}

	// Get current state for progress notification
	state, err := uc.stateUC.GetState(ctx, projectID)
	if err != nil {
		uc.l.Errorf(ctx, "results.usecase.result.handleAnalyzeResult: failed to get state: project_id=%s, error=%v", projectID, err)
		return fmt.Errorf("%w: %v", results.ErrTemporary, err)
	}

	// Get user_id for webhook
	userID, err := uc.stateUC.GetUserID(ctx, projectID)
	if err != nil {
		uc.l.Warnf(ctx, "results.usecase.result.handleAnalyzeResult: failed to get user_id, using empty: project_id=%s, error=%v", projectID, err)
		userID = ""
	}

	// Send progress webhook with two-phase format
	progressReq := uc.buildTwoPhaseProgressRequest(projectID, userID, state)

	if err := uc.webhookUC.NotifyProgress(ctx, progressReq); err != nil {
		uc.l.Warnf(ctx, "results.usecase.result.handleAnalyzeResult: failed to send progress webhook (non-fatal): project_id=%s, error=%v", projectID, err)
	}

	// Check if project is complete (both crawl and analyze phases)
	completed, err := uc.stateUC.CheckCompletion(ctx, projectID)
	if err != nil {
		uc.l.Warnf(ctx, "results.usecase.result.handleAnalyzeResult: failed to check completion: project_id=%s, error=%v", projectID, err)
	} else if completed {
		uc.l.Infof(ctx, "results.usecase.result.handleAnalyzeResult: project completed: project_id=%s", projectID)
		// Send completion notification
		if err := uc.webhookUC.NotifyCompletion(ctx, progressReq); err != nil {
			uc.l.Warnf(ctx, "results.usecase.result.handleAnalyzeResult: failed to send completion webhook: project_id=%s, error=%v", projectID, err)
		}
	}

	return nil
}

// extractAnalyzePayload extracts AnalyzeResultPayload from the result payload
func (uc implUseCase) extractAnalyzePayload(ctx context.Context, payload any) (*results.AnalyzeResultPayload, error) {
	if payload == nil {
		return nil, results.ErrInvalidInput
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		uc.l.Errorf(ctx, "results.usecase.result.extractAnalyzePayload: failed to marshal payload: %v", err)
		return nil, results.ErrInvalidInput
	}

	var analyzePayload results.AnalyzeResultPayload
	if err := json.Unmarshal(jsonData, &analyzePayload); err != nil {
		uc.l.Errorf(ctx, "results.usecase.result.extractAnalyzePayload: failed to unmarshal payload: %v", err)
		return nil, results.ErrInvalidInput
	}

	return &analyzePayload, nil
}

// extractProjectID extracts project_id from job_id in payload
// job_id format for project execution: {projectID}-brand-{index}
func (uc implUseCase) extractProjectID(ctx context.Context, payload any) (string, error) {
	if payload == nil {
		uc.l.Errorf(ctx, "results.usecase.result.extractProjectID: payload is nil")
		return "", results.ErrInvalidInput
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		uc.l.Errorf(ctx, "results.usecase.result.extractProjectID: failed to marshal payload: %v", err)
		return "", results.ErrInvalidInput
	}

	var crawlerContentArray []results.CrawlerContent
	if err := json.Unmarshal(jsonData, &crawlerContentArray); err != nil {
		uc.l.Errorf(ctx, "results.usecase.result.extractProjectID: failed to unmarshal payload: %v", err)
		return "", results.ErrInvalidInput
	} else if len(crawlerContentArray) == 0 {
		uc.l.Errorf(ctx, "results.usecase.result.extractProjectID: empty content array")
		return "", results.ErrInvalidInput
	} else if crawlerContentArray[0].Meta.JobID == "" {
		uc.l.Errorf(ctx, "results.usecase.result.extractProjectID: job_id is empty")
		return "", results.ErrInvalidInput
	}

	if len(crawlerContentArray) == 0 {
		uc.l.Errorf(ctx, "results.usecase.result.extractProjectID: empty content array")
		return "", results.ErrInvalidInput
	}

	jobID := crawlerContentArray[0].Meta.JobID
	if jobID == "" {
		uc.l.Errorf(ctx, "results.usecase.result.extractProjectID: job_id is empty")
		return "", results.ErrInvalidInput
	}

	// Extract project_id from job_id (format: {projectID}-brand-{index})
	// Find the last occurrence of "-brand-" and take everything before it
	const brandSuffix = "-brand-"
	idx := strings.LastIndex(jobID, brandSuffix)
	if idx == -1 {
		// If no "-brand-" suffix, assume job_id is the project_id
		return jobID, nil
	}

	return jobID[:idx], nil
}

// handleWebhookError determines if a webhook error is permanent or temporary
func (uc implUseCase) handleWebhookError(ctx context.Context, jobID, platform string, err error) error {
	// Check if error message indicates a 4xx client error (permanent)
	errMsg := err.Error()

	// 4xx errors are permanent - don't retry
	if contains(errMsg, "client error") || contains(errMsg, "not retrying") {
		uc.l.Errorf(ctx, "results.usecase.result.handleWebhookError: webhook failed with permanent error (4xx): job_id=%s, platform=%s, error=%v",
			jobID, platform, err)
		// Return ErrInvalidInput to signal permanent failure (no retry)
		return results.ErrInvalidInput
	}

	// Check for specific error types from project client
	if errors.Is(err, project.ErrProjectUnavailable) {
		// 5xx or network errors - temporary, should retry
		uc.l.Errorf(ctx, "results.usecase.result.handleWebhookError: webhook failed with temporary error (5xx/network): job_id=%s, platform=%s, error=%v",
			jobID, platform, err)
		return results.ErrTemporary
	}

	if errors.Is(err, project.ErrProjectTimeout) {
		// Timeout - temporary, should retry
		uc.l.Errorf(ctx, "results.usecase.result.handleWebhookError: webhook failed with timeout: job_id=%s, platform=%s, error=%v",
			jobID, platform, err)
		return results.ErrTemporary
	}

	if errors.Is(err, project.ErrProjectUnauthorized) {
		// Unauthorized - permanent, don't retry
		uc.l.Errorf(ctx, "results.usecase.result.handleWebhookError: webhook failed with unauthorized error: job_id=%s, platform=%s, error=%v",
			jobID, platform, err)
		return results.ErrInvalidInput
	}

	// Default: treat unknown errors as temporary (safer to retry)
	uc.l.Errorf(ctx, "results.usecase.result.handleWebhookError: webhook failed with unknown error (treating as temporary): job_id=%s, platform=%s, error=%v",
		jobID, platform, err)
	return results.ErrTemporary
}

// contains checks if a string contains a substring (case-insensitive helper)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func (uc implUseCase) buildCallbackRequest(ctx context.Context, res models.CrawlerResult) (project.CallbackRequest, error) {
	// Determine status based on success field
	status := "failed"
	if res.Success {
		status = "success"
	}

	// Build payload
	payload := project.CallbackPayload{}

	// Extract job_id and platform from payload (will be extracted in extractContent)
	var jobID, platform string

	// If successful, extract content from result payload
	if res.Success && res.Payload != nil {
		content, extractedJobID, extractedPlatform, err := uc.extractContent(ctx, res.Payload)
		if err != nil {
			// Error already logged in extractContent
			return project.CallbackRequest{}, results.ErrExtractContentFailed
		}
		payload.Content = content
		jobID = extractedJobID
		platform = extractedPlatform
	} else {
		// For failed cases, try to extract job_id and platform from payload if possible
		jobID, platform = uc.tryExtractMetadata(ctx, res.Payload)
		if jobID == "" {
			jobID = "unknown"
		}
		if platform == "" {
			platform = "unknown"
		}
	}

	return project.CallbackRequest{
		JobID:    jobID,
		Status:   status,
		Platform: platform,
		Payload:  payload,
	}, nil
}

// tryExtractMetadata attempts to extract job_id and platform from payload
func (uc implUseCase) tryExtractMetadata(ctx context.Context, payload any) (jobID, platform string) {
	if payload == nil {
		return "", ""
	}

	// Try to unmarshal and extract first item's meta
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return "", ""
	}

	// payload is already the array of content items from CrawlerResult.Payload
	var crawlerContentArray []results.CrawlerContent
	if err := json.Unmarshal(jsonData, &crawlerContentArray); err != nil {
		return "", ""
	}

	if len(crawlerContentArray) > 0 {
		return crawlerContentArray[0].Meta.JobID, crawlerContentArray[0].Meta.Platform
	}

	return "", ""
}

func (uc implUseCase) extractContent(ctx context.Context, payload any) ([]project.Content, string, string, error) {
	// Step 1: Marshal to JSON (handles the generic 'any' type)
	jsonData, err := json.Marshal(payload)
	if err != nil {
		uc.l.Errorf(ctx, "results.usecase.result.extractContent: failed to marshal payload to JSON: error=%v", err)
		return nil, "", "", results.ErrMarshalPayloadFailed
	}

	// Step 2: Unmarshal to typed CrawlerContent array
	// Note: payload is already the array of content items from CrawlerResult.Payload,
	// not the full message with success/payload wrapper
	var crawlerContentArray []results.CrawlerContent
	if err := json.Unmarshal(jsonData, &crawlerContentArray); err != nil {
		// Truncate raw message for logging (max 500 chars)
		rawMsg := string(jsonData)
		if len(rawMsg) > 500 {
			rawMsg = rawMsg[:500] + "... (truncated)"
		}
		uc.l.Errorf(ctx, "results.usecase.result.extractContent: failed to unmarshal to CrawlerContent array: raw_message=%s, error=%v",
			rawMsg, err)
		return nil, "", "", results.ErrUnmarshalContentFailed
	}

	// Extract job_id and platform from first item (if available)
	var jobID, platform string
	if len(crawlerContentArray) > 0 {
		jobID = crawlerContentArray[0].Meta.JobID
		platform = crawlerContentArray[0].Meta.Platform
	}

	// Step 3: Map crawler content to project content
	content, err := uc.mapCrawlerContentToProjectContent(ctx, jobID, platform, crawlerContentArray)
	if err != nil {
		return nil, jobID, platform, err
	}

	// Log success with content count
	uc.l.Infof(ctx, "Successfully extracted content: job_id=%s, platform=%s, content_count=%d",
		jobID, platform, len(content))

	return content, jobID, platform, nil
}

// mapCrawlerContentToProjectContent converts a slice of CrawlerContent to a slice of project.Content
func (uc implUseCase) mapCrawlerContentToProjectContent(ctx context.Context, jobID, platform string, crawlerContent []results.CrawlerContent) ([]project.Content, error) {
	projectContent := make([]project.Content, 0, len(crawlerContent))
	for i, crawlerItem := range crawlerContent {
		// Validate required fields
		if err := uc.validateCrawlerContent(ctx, jobID, platform, i, crawlerItem); err != nil {
			uc.l.Errorf(ctx, "results.usecase.result.mapCrawlerContentToProjectContent: validation failed for content at index %d: %v", i, err)
			return nil, err
		}

		// Map crawler content to project content
		projectItem, err := uc.mapCrawlerContentItemToProjectContent(ctx, jobID, platform, crawlerItem)
		if err != nil {
			uc.l.Errorf(ctx, "results.usecase.result.mapCrawlerContentToProjectContent: failed to map content at index %d (job_id=%s, platform=%s, content_id=%s): %v", i, crawlerItem.Meta.JobID, crawlerItem.Meta.Platform, crawlerItem.Meta.ID, err)
			return nil, err
		}

		projectContent = append(projectContent, projectItem)
	}

	return projectContent, nil
}

// validateCrawlerContent validates that required fields are present in a crawler content item
func (uc implUseCase) validateCrawlerContent(ctx context.Context, jobID, platform string, index int, content results.CrawlerContent) error {
	missingFields := []string{}

	if content.Meta.JobID == "" {
		missingFields = append(missingFields, "job_id")
	}
	if content.Meta.Platform == "" {
		missingFields = append(missingFields, "platform")
	}
	if content.Meta.ID == "" {
		missingFields = append(missingFields, "content_id (meta.id)")
	}

	if len(missingFields) > 0 {
		uc.l.Errorf(ctx, "Validation failed for content at index %d: job_id=%s, platform=%s, missing_fields=%v",
			index, jobID, platform, missingFields)
		return results.ErrMissingRequiredFields
	}

	return nil
}

// mapCrawlerContentItemToProjectContent converts a CrawlerContent to a project.Content
func (uc implUseCase) mapCrawlerContentItemToProjectContent(ctx context.Context, jobID, platform string, cc results.CrawlerContent) (project.Content, error) {
	// Map meta with timestamp parsing
	meta, err := uc.mapCrawlerMetaToProjectMeta(ctx, jobID, platform, cc.Meta.ID, cc.Meta)
	if err != nil {
		uc.l.Errorf(ctx, "results.usecase.result.mapCrawlerContentItemToProjectContent: failed to map meta: %v", err)
		return project.Content{}, results.ErrMapMetaFailed
	}

	// Map content data
	contentData, err := uc.mapCrawlerContentDataToProjectContentData(ctx, jobID, platform, cc.Meta.ID, cc.Content)
	if err != nil {
		uc.l.Errorf(ctx, "results.usecase.result.mapCrawlerContentItemToProjectContent: failed to map content data: %v", err)
		return project.Content{}, results.ErrMapContentDataFailed
	}

	// Map interaction with timestamp parsing
	interaction, err := uc.mapCrawlerInteractionToProjectInteraction(ctx, jobID, platform, cc.Meta.ID, cc.Interaction)
	if err != nil {
		uc.l.Errorf(ctx, "results.usecase.result.mapCrawlerContentItemToProjectContent: failed to map interaction: %v", err)
		return project.Content{}, results.ErrMapInteractionFailed
	}

	// Map author
	author := uc.mapCrawlerAuthorToProjectAuthor(cc.Author)

	// Log YouTube-specific author fields for debugging
	if cc.Author.Country != nil || cc.Author.TotalViewCount != nil {
		uc.l.Debugf(ctx, "Mapping YouTube author fields: job_id=%s, platform=%s, content_id=%s, has_country=%v, has_total_view_count=%v",
			jobID, platform, cc.Meta.ID, cc.Author.Country != nil, cc.Author.TotalViewCount != nil)
	}

	// Map comments with timestamp parsing
	comments, err := uc.mapCrawlerCommentsToProjectComments(ctx, jobID, platform, cc.Meta.ID, cc.Comments)
	if err != nil {
		uc.l.Errorf(ctx, "results.usecase.result.mapCrawlerContentItemToProjectContent: failed to map comments: %v", err)
		return project.Content{}, results.ErrMapCommentsFailed
	}

	return project.Content{
		Meta:        meta,
		Content:     contentData,
		Interaction: interaction,
		Author:      author,
		Comments:    comments,
	}, nil
}

// mapCrawlerMetaToProjectMeta converts CrawlerContentMeta to project.ContentMeta
func (uc implUseCase) mapCrawlerMetaToProjectMeta(ctx context.Context, jobID, platform, contentID string, cm results.CrawlerContentMeta) (project.ContentMeta, error) {
	// Parse timestamps
	crawledAt, err := parseTimestamp(cm.CrawledAt)
	if err != nil {
		uc.l.Errorf(ctx, "Failed to parse crawled_at timestamp: job_id=%s, platform=%s, content_id=%s, field=crawled_at, value=%s, error=%v",
			jobID, platform, contentID, cm.CrawledAt, err)
		return project.ContentMeta{}, results.ErrInvalidTimestamp
	}

	publishedAt, err := parseTimestamp(cm.PublishedAt)
	if err != nil {
		uc.l.Errorf(ctx, "Failed to parse published_at timestamp: job_id=%s, platform=%s, content_id=%s, field=published_at, value=%s, error=%v",
			jobID, platform, contentID, cm.PublishedAt, err)
		return project.ContentMeta{}, results.ErrInvalidTimestamp
	}

	return project.ContentMeta{
		ID:              cm.ID,
		Platform:        cm.Platform,
		JobID:           cm.JobID,
		CrawledAt:       crawledAt,
		PublishedAt:     publishedAt,
		Permalink:       cm.Permalink,
		KeywordSource:   cm.KeywordSource,
		Lang:            cm.Lang,
		Region:          cm.Region,
		PipelineVersion: cm.PipelineVersion,
		FetchStatus:     cm.FetchStatus,
		FetchError:      cm.FetchError,
	}, nil
}

// mapCrawlerContentDataToProjectContentData converts CrawlerContentData to project.ContentData
func (uc implUseCase) mapCrawlerContentDataToProjectContentData(ctx context.Context, jobID, platform, contentID string, cc results.CrawlerContentData) (project.ContentData, error) {
	var media *project.ContentMedia
	if cc.Media != nil {
		var downloadedAt time.Time
		var err error
		if cc.Media.DownloadedAt != "" {
			downloadedAt, err = parseTimestamp(cc.Media.DownloadedAt)
			if err != nil {
				uc.l.Errorf(ctx, "Failed to parse media.downloaded_at timestamp: job_id=%s, platform=%s, content_id=%s, field=media.downloaded_at, value=%s, error=%v",
					jobID, platform, contentID, cc.Media.DownloadedAt, err)
				return project.ContentData{}, results.ErrInvalidTimestamp
			}
		}

		media = &project.ContentMedia{
			Type:         cc.Media.Type,
			VideoPath:    cc.Media.VideoPath,
			AudioPath:    cc.Media.AudioPath,
			DownloadedAt: downloadedAt,
		}
	}

	// Log YouTube-specific field presence for debugging
	if cc.Title != nil {
		uc.l.Debugf(ctx, "Mapping YouTube title field: job_id=%s, platform=%s, content_id=%s, has_title=true",
			jobID, platform, contentID)
	}

	return project.ContentData{
		Text:          cc.Text,
		Duration:      cc.Duration,
		Hashtags:      cc.Hashtags,
		SoundName:     cc.SoundName,
		Category:      cc.Category,
		Title:         cc.Title, // YouTube only
		Media:         media,
		Transcription: cc.Transcription,
	}, nil
}

// mapCrawlerInteractionToProjectInteraction converts CrawlerContentInteraction to project.ContentInteraction
func (uc implUseCase) mapCrawlerInteractionToProjectInteraction(ctx context.Context, jobID, platform, contentID string, ci results.CrawlerContentInteraction) (project.ContentInteraction, error) {
	// Parse updated_at timestamp (optional - can be null/empty)
	var updatedAt time.Time
	if ci.UpdatedAt != "" {
		var err error
		updatedAt, err = parseTimestamp(ci.UpdatedAt)
		if err != nil {
			uc.l.Errorf(ctx, "Failed to parse updated_at timestamp: job_id=%s, platform=%s, content_id=%s, field=updated_at, value=%s, error=%v",
				jobID, platform, contentID, ci.UpdatedAt, err)
			return project.ContentInteraction{}, results.ErrInvalidTimestamp
		}
	}

	return project.ContentInteraction{
		Views:          ci.Views,
		Likes:          ci.Likes,
		CommentsCount:  ci.CommentsCount,
		Shares:         ci.Shares,
		Saves:          ci.Saves,
		EngagementRate: ci.EngagementRate,
		UpdatedAt:      updatedAt,
	}, nil
}

// mapCrawlerAuthorToProjectAuthor converts CrawlerContentAuthor to project.ContentAuthor
func (uc implUseCase) mapCrawlerAuthorToProjectAuthor(ca results.CrawlerContentAuthor) project.ContentAuthor {
	return project.ContentAuthor{
		ID:             ca.ID,
		Name:           ca.Name,
		Username:       ca.Username,
		Followers:      ca.Followers,
		Following:      ca.Following,
		Likes:          ca.Likes,
		Videos:         ca.Videos,
		IsVerified:     ca.IsVerified,
		Bio:            ca.Bio,
		AvatarURL:      ca.AvatarURL,
		ProfileURL:     ca.ProfileURL,
		Country:        ca.Country,        // YouTube only
		TotalViewCount: ca.TotalViewCount, // YouTube only
	}
}

// mapCrawlerCommentsToProjectComments converts []CrawlerComment to []project.Comment
func (uc implUseCase) mapCrawlerCommentsToProjectComments(ctx context.Context, jobID, platform, contentID string, comments []results.CrawlerComment) ([]project.Comment, error) {
	if len(comments) == 0 {
		uc.l.Infof(ctx, "results.usecase.result.mapCrawlerCommentsToProjectComments: no comments found")
		return nil, nil
	}

	projectComments := make([]project.Comment, 0, len(comments))
	for i, cc := range comments {
		// Parse published_at timestamp
		publishedAt, err := parseTimestamp(cc.PublishedAt)
		if err != nil {
			uc.l.Errorf(ctx, "results.usecase.result.mapCrawlerCommentsToProjectComments: failed to parse comment published_at timestamp: job_id=%s, platform=%s, content_id=%s, comment_index=%d, comment_id=%s, field=published_at, value=%s, error=%v",
				jobID, platform, contentID, i, cc.ID, cc.PublishedAt, err)
			return nil, results.ErrInvalidTimestamp
		}

		projectComments = append(projectComments, project.Comment{
			ID:       cc.ID,
			ParentID: cc.ParentID,
			PostID:   cc.PostID,
			User: project.CommentUser{
				ID:        cc.User.ID,
				Name:      cc.User.Name,
				AvatarURL: cc.User.AvatarURL,
			},
			Text:         cc.Text,
			Likes:        cc.Likes,
			RepliesCount: cc.RepliesCount,
			PublishedAt:  publishedAt,
			IsAuthor:     cc.IsAuthor,
			Media:        cc.Media,
			IsFavorited:  cc.IsFavorited, // YouTube only
		})

		// Log YouTube-specific comment field for debugging
		if cc.IsFavorited {
			uc.l.Debugf(ctx, "results.usecase.result.mapCrawlerCommentsToProjectComments: mapping YouTube favorited comment: job_id=%s, platform=%s, content_id=%s, comment_id=%s, is_favorited=true",
				jobID, platform, contentID, cc.ID)
		}
	}

	return projectComments, nil
}

// parseTimestamp converts a timestamp string to time.Time
// Supports multiple formats: RFC3339, RFC3339Nano, and datetime without timezone
func parseTimestamp(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, results.ErrEmptyTimestamp
	}

	// Try RFC3339 first (most common)
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}

	// Try RFC3339Nano (with nanoseconds and timezone)
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t, nil
	}

	// Try datetime without timezone (e.g., "2025-12-02T21:30:06.704383")
	if t, err := time.Parse("2006-01-02T15:04:05.999999", s); err == nil {
		return t, nil
	}

	// Try datetime without timezone and without fractional seconds
	if t, err := time.Parse("2006-01-02T15:04:05", s); err == nil {
		return t, nil
	}

	return time.Time{}, results.ErrUnsupportedTimestampFormat
}
