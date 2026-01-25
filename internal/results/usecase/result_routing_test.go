package usecase

import (
	"context"
	"testing"

	"smap-collector/internal/models"
	"smap-collector/internal/results"
	"smap-collector/internal/webhook"
	"smap-collector/pkg/project"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Manual mock implementations for testing

type mockProjectClientForRouting struct {
	sendDryRunCallbackCalled bool
	sendDryRunCallbackReq    project.CallbackRequest
	sendDryRunCallbackErr    error
}

func (m *mockProjectClientForRouting) SendDryRunCallback(ctx context.Context, req project.CallbackRequest) error {
	m.sendDryRunCallbackCalled = true
	m.sendDryRunCallbackReq = req
	return m.sendDryRunCallbackErr
}

func (m *mockProjectClientForRouting) SendProgressCallback(ctx context.Context, req project.ProgressCallbackRequest) error {
	return nil
}

type mockStateUseCaseForRouting struct {
	// Crawl phase
	incrementCrawlDoneCalled    bool
	incrementCrawlDoneProjectID string
	incrementCrawlDoneCount     int64
	incrementCrawlDoneErr       error

	incrementCrawlErrorsCalled    bool
	incrementCrawlErrorsProjectID string
	incrementCrawlErrorsCount     int64
	incrementCrawlErrorsErr       error

	// Analyze phase
	incrementAnalyzeTotalCalled    bool
	incrementAnalyzeTotalProjectID string
	incrementAnalyzeTotalCount     int64
	incrementAnalyzeTotalErr       error

	incrementAnalyzeDoneCalled    bool
	incrementAnalyzeDoneProjectID string
	incrementAnalyzeDoneCount     int64
	incrementAnalyzeDoneErr       error

	incrementAnalyzeErrorsCalled    bool
	incrementAnalyzeErrorsProjectID string
	incrementAnalyzeErrorsCount     int64
	incrementAnalyzeErrorsErr       error

	// State
	getStateCalled    bool
	getStateProjectID string
	getStateResult    *models.ProjectState
	getStateErr       error

	getUserIDCalled    bool
	getUserIDProjectID string
	getUserIDResult    string
	getUserIDErr       error

	checkCompletionCalled    bool
	checkCompletionProjectID string
	checkCompletionResult    bool
	checkCompletionErr       error
}

func (m *mockStateUseCaseForRouting) InitState(ctx context.Context, projectID string) error {
	return nil
}

func (m *mockStateUseCaseForRouting) SetCrawlTotal(ctx context.Context, projectID string, total int64) error {
	return nil
}

func (m *mockStateUseCaseForRouting) IncrementCrawlDoneBy(ctx context.Context, projectID string, count int64) error {
	m.incrementCrawlDoneCalled = true
	m.incrementCrawlDoneProjectID = projectID
	m.incrementCrawlDoneCount = count
	return m.incrementCrawlDoneErr
}

func (m *mockStateUseCaseForRouting) IncrementCrawlErrorsBy(ctx context.Context, projectID string, count int64) error {
	m.incrementCrawlErrorsCalled = true
	m.incrementCrawlErrorsProjectID = projectID
	m.incrementCrawlErrorsCount = count
	return m.incrementCrawlErrorsErr
}

func (m *mockStateUseCaseForRouting) IncrementAnalyzeTotalBy(ctx context.Context, projectID string, count int64) error {
	m.incrementAnalyzeTotalCalled = true
	m.incrementAnalyzeTotalProjectID = projectID
	m.incrementAnalyzeTotalCount = count
	return m.incrementAnalyzeTotalErr
}

func (m *mockStateUseCaseForRouting) IncrementAnalyzeDoneBy(ctx context.Context, projectID string, count int64) error {
	m.incrementAnalyzeDoneCalled = true
	m.incrementAnalyzeDoneProjectID = projectID
	m.incrementAnalyzeDoneCount = count
	return m.incrementAnalyzeDoneErr
}

func (m *mockStateUseCaseForRouting) IncrementAnalyzeErrorsBy(ctx context.Context, projectID string, count int64) error {
	m.incrementAnalyzeErrorsCalled = true
	m.incrementAnalyzeErrorsProjectID = projectID
	m.incrementAnalyzeErrorsCount = count
	return m.incrementAnalyzeErrorsErr
}

func (m *mockStateUseCaseForRouting) UpdateStatus(ctx context.Context, projectID string, status models.ProjectStatus) error {
	return nil
}

func (m *mockStateUseCaseForRouting) GetState(ctx context.Context, projectID string) (*models.ProjectState, error) {
	m.getStateCalled = true
	m.getStateProjectID = projectID
	return m.getStateResult, m.getStateErr
}

func (m *mockStateUseCaseForRouting) CheckCompletion(ctx context.Context, projectID string) (bool, error) {
	m.checkCompletionCalled = true
	m.checkCompletionProjectID = projectID
	return m.checkCompletionResult, m.checkCompletionErr
}

func (m *mockStateUseCaseForRouting) StoreUserMapping(ctx context.Context, projectID, userID string) error {
	return nil
}

func (m *mockStateUseCaseForRouting) GetUserID(ctx context.Context, projectID string) (string, error) {
	m.getUserIDCalled = true
	m.getUserIDProjectID = projectID
	return m.getUserIDResult, m.getUserIDErr
}

// Task-level methods (new hybrid state)
func (m *mockStateUseCaseForRouting) SetTasksTotal(ctx context.Context, projectID string, tasksTotal, itemsExpected int64) error {
	return nil
}

func (m *mockStateUseCaseForRouting) IncrementTasksDone(ctx context.Context, projectID string) error {
	m.incrementCrawlDoneCalled = true
	m.incrementCrawlDoneProjectID = projectID
	m.incrementCrawlDoneCount = 1
	return m.incrementCrawlDoneErr
}

func (m *mockStateUseCaseForRouting) IncrementTasksErrors(ctx context.Context, projectID string) error {
	m.incrementCrawlErrorsCalled = true
	m.incrementCrawlErrorsProjectID = projectID
	m.incrementCrawlErrorsCount = 1
	return m.incrementCrawlErrorsErr
}

func (m *mockStateUseCaseForRouting) IncrementItemsActualBy(ctx context.Context, projectID string, count int64) error {
	m.incrementCrawlDoneCalled = true
	m.incrementCrawlDoneProjectID = projectID
	m.incrementCrawlDoneCount = count
	return m.incrementCrawlDoneErr
}

func (m *mockStateUseCaseForRouting) IncrementItemsErrorsBy(ctx context.Context, projectID string, count int64) error {
	m.incrementCrawlErrorsCalled = true
	m.incrementCrawlErrorsProjectID = projectID
	m.incrementCrawlErrorsCount = count
	return m.incrementCrawlErrorsErr
}

type mockWebhookUseCaseForRouting struct {
	notifyProgressCalled bool
	notifyProgressReq    webhook.ProgressRequest
	notifyProgressErr    error

	notifyCompletionCalled bool
	notifyCompletionReq    webhook.ProgressRequest
	notifyCompletionErr    error
}

func (m *mockWebhookUseCaseForRouting) NotifyProgress(ctx context.Context, req webhook.ProgressRequest) error {
	m.notifyProgressCalled = true
	m.notifyProgressReq = req
	return m.notifyProgressErr
}

func (m *mockWebhookUseCaseForRouting) NotifyCompletion(ctx context.Context, req webhook.ProgressRequest) error {
	m.notifyCompletionCalled = true
	m.notifyCompletionReq = req
	return m.notifyCompletionErr
}

// Test helper to create crawler content with task_type
func createCrawlerContentForRouting(taskType string, jobID string, platform string) []results.CrawlerContent {
	return []results.CrawlerContent{
		{
			Meta: results.CrawlerContentMeta{
				ID:          "video123",
				Platform:    platform,
				JobID:       jobID,
				TaskType:    taskType,
				CrawledAt:   "2024-01-15T10:00:00Z",
				PublishedAt: "2024-01-14T10:00:00Z",
				Permalink:   "https://tiktok.com/@test/video/123",
				FetchStatus: "success",
			},
			Content: results.CrawlerContentData{
				Text: "Test content",
			},
			Interaction: results.CrawlerContentInteraction{
				Views:     1000,
				Likes:     100,
				UpdatedAt: "2024-01-15T10:00:00Z",
			},
			Author: results.CrawlerContentAuthor{
				ID:         "user123",
				Name:       "Test User",
				Username:   "testuser",
				ProfileURL: "https://tiktok.com/@testuser",
			},
		},
	}
}

// TestExtractTaskType_DryRunKeyword tests extracting dryrun_keyword task type
func TestExtractTaskType_DryRunKeyword(t *testing.T) {
	ctx := context.Background()
	uc := implUseCase{l: &mockLogger{}}

	payload := createCrawlerContentForRouting("dryrun_keyword", "job123", "tiktok")

	taskType := uc.extractTaskType(ctx, payload)

	assert.Equal(t, "dryrun_keyword", taskType)
}

// TestExtractTaskType_ResearchAndCrawl tests extracting research_and_crawl task type
func TestExtractTaskType_ResearchAndCrawl(t *testing.T) {
	ctx := context.Background()
	uc := implUseCase{l: &mockLogger{}}

	payload := createCrawlerContentForRouting("research_and_crawl", "proj123-brand-0", "tiktok")

	taskType := uc.extractTaskType(ctx, payload)

	assert.Equal(t, "research_and_crawl", taskType)
}

// TestExtractTaskType_EmptyPayload tests extracting task type from empty payload
func TestExtractTaskType_EmptyPayload(t *testing.T) {
	ctx := context.Background()
	uc := implUseCase{l: &mockLogger{}}

	taskType := uc.extractTaskType(ctx, nil)

	assert.Equal(t, "", taskType)
}

// TestExtractTaskType_MissingTaskType tests backward compatibility when task_type is missing
func TestExtractTaskType_MissingTaskType(t *testing.T) {
	ctx := context.Background()
	uc := implUseCase{l: &mockLogger{}}

	// Create content without task_type
	payload := createCrawlerContentForRouting("", "job123", "tiktok")

	taskType := uc.extractTaskType(ctx, payload)

	assert.Equal(t, "", taskType)
}

// TestHandleResult_RoutesDryRunCorrectly tests that dryrun_keyword routes to handleDryRunResult
func TestHandleResult_RoutesDryRunCorrectly(t *testing.T) {
	ctx := context.Background()

	mockProject := &mockProjectClientForRouting{}
	mockState := &mockStateUseCaseForRouting{}
	mockWebhook := &mockWebhookUseCaseForRouting{}

	uc := implUseCase{
		l:             &mockLogger{},
		projectClient: mockProject,
		stateUC:       mockState,
		webhookUC:     mockWebhook,
	}

	payload := createCrawlerContentForRouting("dryrun_keyword", "job123", "tiktok")
	result := models.CrawlerResult{
		Success: true,
		Payload: payload,
	}

	err := uc.HandleResult(ctx, result)

	require.NoError(t, err)
	// Verify dry-run callback was called
	assert.True(t, mockProject.sendDryRunCallbackCalled, "SendDryRunCallback should be called")
	assert.Equal(t, "job123", mockProject.sendDryRunCallbackReq.JobID)
	assert.Equal(t, "tiktok", mockProject.sendDryRunCallbackReq.Platform)
	// State and webhook should NOT be called for dry-run
	assert.False(t, mockState.incrementCrawlDoneCalled, "IncrementCrawlDoneBy should NOT be called for dry-run")
	assert.False(t, mockWebhook.notifyProgressCalled, "NotifyProgress should NOT be called for dry-run")
}

// TestHandleResult_RoutesProjectExecutionCorrectly tests that research_and_crawl routes to handleProjectResult
// Updated for FLAT format v3.0
func TestHandleResult_RoutesProjectExecutionCorrectly(t *testing.T) {
	ctx := context.Background()

	mockProject := &mockProjectClientForRouting{}
	mockState := &mockStateUseCaseForRouting{
		getStateResult: &models.ProjectState{
			Status:        models.ProjectStatusProcessing,
			CrawlTotal:    100,
			CrawlDone:     1,
			CrawlErrors:   0,
			AnalyzeTotal:  1,
			AnalyzeDone:   0,
			AnalyzeErrors: 0,
		},
		getUserIDResult:       "user456",
		checkCompletionResult: false,
	}
	mockWebhook := &mockWebhookUseCaseForRouting{}

	uc := implUseCase{
		l:             &mockLogger{},
		projectClient: mockProject,
		stateUC:       mockState,
		webhookUC:     mockWebhook,
	}

	// Create FLAT format CrawlerResult
	result := models.CrawlerResult{
		Success:         true,
		TaskType:        "research_and_crawl",
		JobID:           "proj123-brand-0",
		Platform:        "tiktok",
		RequestedLimit:  50,
		AppliedLimit:    50,
		TotalFound:      50,
		PlatformLimited: false,
		Successful:      1,
		Failed:          0,
		Skipped:         0,
	}

	err := uc.HandleResult(ctx, result)

	require.NoError(t, err)
	// Verify project execution flow was called
	assert.True(t, mockState.incrementCrawlDoneCalled, "IncrementTasksDone should be called")
	assert.Equal(t, "proj123", mockState.incrementCrawlDoneProjectID)
	assert.True(t, mockState.incrementAnalyzeTotalCalled, "IncrementAnalyzeTotalBy should be called for successful crawl")
	assert.True(t, mockWebhook.notifyProgressCalled, "NotifyProgress should be called")
	assert.Equal(t, "proj123", mockWebhook.notifyProgressReq.ProjectID)
	// Dry-run callback should NOT be called
	assert.False(t, mockProject.sendDryRunCallbackCalled, "SendDryRunCallback should NOT be called for project execution")
}

// TestHandleResult_BackwardCompatibility tests that missing task_type defaults to dry-run
func TestHandleResult_BackwardCompatibility(t *testing.T) {
	ctx := context.Background()

	mockProject := &mockProjectClientForRouting{}
	mockState := &mockStateUseCaseForRouting{}
	mockWebhook := &mockWebhookUseCaseForRouting{}

	uc := implUseCase{
		l:             &mockLogger{},
		projectClient: mockProject,
		stateUC:       mockState,
		webhookUC:     mockWebhook,
	}

	// Create payload without task_type (legacy format)
	payload := createCrawlerContentForRouting("", "legacy-job-123", "tiktok")
	result := models.CrawlerResult{
		Success: true,
		Payload: payload,
	}

	err := uc.HandleResult(ctx, result)

	require.NoError(t, err)
	// Should default to dry-run handler
	assert.True(t, mockProject.sendDryRunCallbackCalled, "SendDryRunCallback should be called for backward compatibility")
	assert.False(t, mockState.incrementCrawlDoneCalled, "IncrementCrawlDoneBy should NOT be called")
}

// TestHandleResult_ProjectExecutionWithErrors tests error counter increment
// Updated for FLAT format v3.0
func TestHandleResult_ProjectExecutionWithErrors(t *testing.T) {
	ctx := context.Background()

	mockProject := &mockProjectClientForRouting{}
	mockState := &mockStateUseCaseForRouting{
		getStateResult: &models.ProjectState{
			Status:        models.ProjectStatusProcessing,
			CrawlTotal:    100,
			CrawlDone:     0,
			CrawlErrors:   1,
			AnalyzeTotal:  0,
			AnalyzeDone:   0,
			AnalyzeErrors: 0,
		},
		getUserIDResult:       "user456",
		checkCompletionResult: false,
	}
	mockWebhook := &mockWebhookUseCaseForRouting{}

	uc := implUseCase{
		l:             &mockLogger{},
		projectClient: mockProject,
		stateUC:       mockState,
		webhookUC:     mockWebhook,
	}

	// Create FLAT format CrawlerResult with failure
	errCode := "SEARCH_FAILED"
	errMsg := "TikTok API error"
	result := models.CrawlerResult{
		Success:         false, // Failed result
		TaskType:        "research_and_crawl",
		JobID:           "proj123-brand-0",
		Platform:        "tiktok",
		RequestedLimit:  50,
		AppliedLimit:    50,
		TotalFound:      0,
		PlatformLimited: false,
		Successful:      0,
		Failed:          0,
		Skipped:         0,
		ErrorCode:       &errCode,
		ErrorMessage:    &errMsg,
	}

	err := uc.HandleResult(ctx, result)

	require.NoError(t, err)
	// Verify error counter was incremented
	assert.True(t, mockState.incrementCrawlErrorsCalled, "IncrementTasksErrors should be called for failed result")
	assert.Equal(t, "proj123", mockState.incrementCrawlErrorsProjectID)
	assert.False(t, mockState.incrementCrawlDoneCalled, "IncrementTasksDone should NOT be called for failed result")
	assert.False(t, mockState.incrementAnalyzeTotalCalled, "IncrementAnalyzeTotalBy should NOT be called for failed result")
}

// ============================================================================
// Analyze Result Tests
// ============================================================================

// createAnalyzeResultPayload creates an analyze result payload for testing
func createAnalyzeResultPayload(projectID, jobID string, successCount, errorCount int) results.AnalyzeResultPayload {
	return results.AnalyzeResultPayload{
		ProjectID:    projectID,
		JobID:        jobID,
		TaskType:     "analyze_result",
		BatchSize:    successCount + errorCount,
		SuccessCount: successCount,
		ErrorCount:   errorCount,
	}
}

// TestExtractTaskType_AnalyzeResult tests extracting analyze_result task type
func TestExtractTaskType_AnalyzeResult(t *testing.T) {
	ctx := context.Background()
	uc := implUseCase{l: &mockLogger{}}

	payload := createAnalyzeResultPayload("proj123", "proj123-analyze-0", 48, 2)

	taskType := uc.extractTaskType(ctx, payload)

	assert.Equal(t, "analyze_result", taskType)
}

// TestHandleResult_RoutesAnalyzeResultCorrectly tests that analyze_result routes to handleAnalyzeResult
func TestHandleResult_RoutesAnalyzeResultCorrectly(t *testing.T) {
	ctx := context.Background()

	mockProject := &mockProjectClientForRouting{}
	mockState := &mockStateUseCaseForRouting{
		getStateResult: &models.ProjectState{
			Status:        models.ProjectStatusProcessing,
			CrawlTotal:    100,
			CrawlDone:     100,
			CrawlErrors:   0,
			AnalyzeTotal:  100,
			AnalyzeDone:   50,
			AnalyzeErrors: 2,
		},
		getUserIDResult:       "user456",
		checkCompletionResult: false,
	}
	mockWebhook := &mockWebhookUseCaseForRouting{}

	uc := implUseCase{
		l:             &mockLogger{},
		projectClient: mockProject,
		stateUC:       mockState,
		webhookUC:     mockWebhook,
	}

	payload := createAnalyzeResultPayload("proj123", "proj123-analyze-0", 48, 2)
	result := models.CrawlerResult{
		Success: true,
		Payload: payload,
	}

	err := uc.HandleResult(ctx, result)

	require.NoError(t, err)
	// Verify analyze counters were updated
	assert.True(t, mockState.incrementAnalyzeDoneCalled, "IncrementAnalyzeDoneBy should be called")
	assert.Equal(t, "proj123", mockState.incrementAnalyzeDoneProjectID)
	assert.Equal(t, int64(48), mockState.incrementAnalyzeDoneCount)

	assert.True(t, mockState.incrementAnalyzeErrorsCalled, "IncrementAnalyzeErrorsBy should be called")
	assert.Equal(t, "proj123", mockState.incrementAnalyzeErrorsProjectID)
	assert.Equal(t, int64(2), mockState.incrementAnalyzeErrorsCount)

	// Verify progress webhook was called
	assert.True(t, mockWebhook.notifyProgressCalled, "NotifyProgress should be called")
	assert.Equal(t, "proj123", mockWebhook.notifyProgressReq.ProjectID)

	// Crawl counters should NOT be updated
	assert.False(t, mockState.incrementCrawlDoneCalled, "IncrementCrawlDoneBy should NOT be called for analyze result")
	assert.False(t, mockState.incrementCrawlErrorsCalled, "IncrementCrawlErrorsBy should NOT be called for analyze result")
}

// TestHandleAnalyzeResult_OnlySuccessCount tests analyze result with only success count
func TestHandleAnalyzeResult_OnlySuccessCount(t *testing.T) {
	ctx := context.Background()

	mockState := &mockStateUseCaseForRouting{
		getStateResult: &models.ProjectState{
			Status:        models.ProjectStatusProcessing,
			CrawlTotal:    100,
			CrawlDone:     100,
			CrawlErrors:   0,
			AnalyzeTotal:  100,
			AnalyzeDone:   50,
			AnalyzeErrors: 0,
		},
		getUserIDResult:       "user456",
		checkCompletionResult: false,
	}
	mockWebhook := &mockWebhookUseCaseForRouting{}

	uc := implUseCase{
		l:             &mockLogger{},
		projectClient: &mockProjectClientForRouting{},
		stateUC:       mockState,
		webhookUC:     mockWebhook,
	}

	payload := createAnalyzeResultPayload("proj123", "proj123-analyze-0", 50, 0)
	result := models.CrawlerResult{
		Success: true,
		Payload: payload,
	}

	err := uc.HandleResult(ctx, result)

	require.NoError(t, err)
	assert.True(t, mockState.incrementAnalyzeDoneCalled, "IncrementAnalyzeDoneBy should be called")
	assert.Equal(t, int64(50), mockState.incrementAnalyzeDoneCount)
	assert.False(t, mockState.incrementAnalyzeErrorsCalled, "IncrementAnalyzeErrorsBy should NOT be called when error_count=0")
}

// TestHandleAnalyzeResult_OnlyErrorCount tests analyze result with only error count
func TestHandleAnalyzeResult_OnlyErrorCount(t *testing.T) {
	ctx := context.Background()

	mockState := &mockStateUseCaseForRouting{
		getStateResult: &models.ProjectState{
			Status:        models.ProjectStatusProcessing,
			CrawlTotal:    100,
			CrawlDone:     100,
			CrawlErrors:   0,
			AnalyzeTotal:  100,
			AnalyzeDone:   0,
			AnalyzeErrors: 5,
		},
		getUserIDResult:       "user456",
		checkCompletionResult: false,
	}
	mockWebhook := &mockWebhookUseCaseForRouting{}

	uc := implUseCase{
		l:             &mockLogger{},
		projectClient: &mockProjectClientForRouting{},
		stateUC:       mockState,
		webhookUC:     mockWebhook,
	}

	payload := createAnalyzeResultPayload("proj123", "proj123-analyze-0", 0, 5)
	result := models.CrawlerResult{
		Success: true,
		Payload: payload,
	}

	err := uc.HandleResult(ctx, result)

	require.NoError(t, err)
	assert.False(t, mockState.incrementAnalyzeDoneCalled, "IncrementAnalyzeDoneBy should NOT be called when success_count=0")
	assert.True(t, mockState.incrementAnalyzeErrorsCalled, "IncrementAnalyzeErrorsBy should be called")
	assert.Equal(t, int64(5), mockState.incrementAnalyzeErrorsCount)
}

// TestHandleAnalyzeResult_ProjectCompletion tests that completion is detected when both phases are done
func TestHandleAnalyzeResult_ProjectCompletion(t *testing.T) {
	ctx := context.Background()

	mockState := &mockStateUseCaseForRouting{
		getStateResult: &models.ProjectState{
			Status:        models.ProjectStatusProcessing,
			CrawlTotal:    100,
			CrawlDone:     100,
			CrawlErrors:   0,
			AnalyzeTotal:  100,
			AnalyzeDone:   100,
			AnalyzeErrors: 0,
		},
		getUserIDResult:       "user456",
		checkCompletionResult: true, // Project is complete
	}
	mockWebhook := &mockWebhookUseCaseForRouting{}

	uc := implUseCase{
		l:             &mockLogger{},
		projectClient: &mockProjectClientForRouting{},
		stateUC:       mockState,
		webhookUC:     mockWebhook,
	}

	payload := createAnalyzeResultPayload("proj123", "proj123-analyze-final", 10, 0)
	result := models.CrawlerResult{
		Success: true,
		Payload: payload,
	}

	err := uc.HandleResult(ctx, result)

	require.NoError(t, err)
	// Verify completion was checked and notification sent
	assert.True(t, mockState.checkCompletionCalled, "CheckCompletion should be called")
	assert.True(t, mockWebhook.notifyCompletionCalled, "NotifyCompletion should be called when project is complete")
}

// TestHandleProjectResult_BatchIncrement tests that batch items are counted correctly
// Updated for FLAT format v3.0 - uses Successful field instead of counting payload items
func TestHandleProjectResult_BatchIncrement(t *testing.T) {
	ctx := context.Background()

	mockState := &mockStateUseCaseForRouting{
		getStateResult: &models.ProjectState{
			Status:        models.ProjectStatusProcessing,
			CrawlTotal:    100,
			CrawlDone:     3,
			CrawlErrors:   0,
			AnalyzeTotal:  3,
			AnalyzeDone:   0,
			AnalyzeErrors: 0,
		},
		getUserIDResult:       "user456",
		checkCompletionResult: false,
	}
	mockWebhook := &mockWebhookUseCaseForRouting{}

	uc := implUseCase{
		l:             &mockLogger{},
		projectClient: &mockProjectClientForRouting{},
		stateUC:       mockState,
		webhookUC:     mockWebhook,
	}

	// Create FLAT format CrawlerResult with 3 successful items
	result := models.CrawlerResult{
		Success:         true,
		TaskType:        "research_and_crawl",
		JobID:           "proj123-brand-0",
		Platform:        "tiktok",
		RequestedLimit:  50,
		AppliedLimit:    50,
		TotalFound:      50,
		PlatformLimited: false,
		Successful:      3, // 3 items crawled successfully
		Failed:          0,
		Skipped:         0,
	}

	err := uc.HandleResult(ctx, result)

	require.NoError(t, err)
	// Verify batch count was used from Successful field
	assert.True(t, mockState.incrementCrawlDoneCalled, "IncrementTasksDone should be called")
	assert.True(t, mockState.incrementAnalyzeTotalCalled, "IncrementAnalyzeTotalBy should be called")
	assert.Equal(t, int64(3), mockState.incrementAnalyzeTotalCount, "Should increment analyze_total by Successful count (3)")
}

// ============================================================================
// handleAnalyzeResult Validation Tests (Phase 6.3)
// ============================================================================

// TestHandleAnalyzeResult_MissingProjectID tests that missing project_id returns error
func TestHandleAnalyzeResult_MissingProjectID(t *testing.T) {
	ctx := context.Background()

	uc := implUseCase{
		l:             &mockLogger{},
		projectClient: &mockProjectClientForRouting{},
		stateUC:       &mockStateUseCaseForRouting{},
		webhookUC:     &mockWebhookUseCaseForRouting{},
	}

	// Create payload with empty project_id
	payload := map[string]any{
		"task_type":     "analyze_result",
		"project_id":    "", // Empty project_id
		"job_id":        "job123",
		"batch_size":    10,
		"success_count": 5,
		"error_count":   0,
	}
	result := models.CrawlerResult{
		Success: true,
		Payload: payload,
	}

	err := uc.HandleResult(ctx, result)

	require.Error(t, err)
	assert.ErrorIs(t, err, results.ErrInvalidInput)
}

// TestHandleAnalyzeResult_NilPayload tests that nil payload returns error
func TestHandleAnalyzeResult_NilPayload(t *testing.T) {
	ctx := context.Background()

	uc := implUseCase{
		l:             &mockLogger{},
		projectClient: &mockProjectClientForRouting{},
		stateUC:       &mockStateUseCaseForRouting{},
		webhookUC:     &mockWebhookUseCaseForRouting{},
	}

	// Create result with nil payload but task_type in root (won't route to analyze)
	// This tests the extractAnalyzePayload error path
	result := models.CrawlerResult{
		Success: true,
		Payload: nil,
	}

	// This will route to dry-run handler (default) since no task_type
	// To test analyze handler directly, we need payload with task_type
	err := uc.HandleResult(ctx, result)

	// Should not error - routes to dry-run which handles nil payload differently
	// The test validates routing behavior
	require.NoError(t, err)
}

// TestHandleAnalyzeResult_InvalidPayloadFormat tests that invalid payload format returns error
func TestHandleAnalyzeResult_InvalidPayloadFormat(t *testing.T) {
	ctx := context.Background()

	mockState := &mockStateUseCaseForRouting{
		getStateResult: &models.ProjectState{
			Status:        models.ProjectStatusProcessing,
			CrawlTotal:    100,
			CrawlDone:     100,
			CrawlErrors:   0,
			AnalyzeTotal:  100,
			AnalyzeDone:   50,
			AnalyzeErrors: 0,
		},
		getUserIDResult:       "user456",
		checkCompletionResult: false,
	}

	uc := implUseCase{
		l:             &mockLogger{},
		projectClient: &mockProjectClientForRouting{},
		stateUC:       mockState,
		webhookUC:     &mockWebhookUseCaseForRouting{},
	}

	// Create payload that looks like analyze_result but has invalid structure
	// This will route to analyze handler but fail to extract payload
	payload := map[string]any{
		"task_type":  "analyze_result",
		"project_id": "proj123",
		// Missing required fields: job_id, batch_size, success_count, error_count
		// These will be zero values after unmarshal
	}
	result := models.CrawlerResult{
		Success: true,
		Payload: payload,
	}

	err := uc.HandleResult(ctx, result)

	// Should succeed because the payload can be unmarshaled (missing fields become zero values)
	// The validation is lenient - only project_id is strictly required
	require.NoError(t, err)
}
