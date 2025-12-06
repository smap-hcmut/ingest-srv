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
	incrementDoneCalled    bool
	incrementDoneProjectID string
	incrementDoneErr       error

	incrementErrorsCalled    bool
	incrementErrorsProjectID string
	incrementErrorsErr       error

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

func (m *mockStateUseCaseForRouting) UpdateTotal(ctx context.Context, projectID string, total int64) error {
	return nil
}

func (m *mockStateUseCaseForRouting) IncrementDone(ctx context.Context, projectID string) error {
	m.incrementDoneCalled = true
	m.incrementDoneProjectID = projectID
	return m.incrementDoneErr
}

func (m *mockStateUseCaseForRouting) IncrementErrors(ctx context.Context, projectID string) error {
	m.incrementErrorsCalled = true
	m.incrementErrorsProjectID = projectID
	return m.incrementErrorsErr
}

func (m *mockStateUseCaseForRouting) UpdateStatus(ctx context.Context, projectID string, status models.ProjectStatus) error {
	return nil
}

func (m *mockStateUseCaseForRouting) GetState(ctx context.Context, projectID string) (*models.ProjectState, error) {
	m.getStateCalled = true
	m.getStateProjectID = projectID
	return m.getStateResult, m.getStateErr
}

func (m *mockStateUseCaseForRouting) CheckAndUpdateCompletion(ctx context.Context, projectID string) (bool, error) {
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
	assert.False(t, mockState.incrementDoneCalled, "IncrementDone should NOT be called for dry-run")
	assert.False(t, mockWebhook.notifyProgressCalled, "NotifyProgress should NOT be called for dry-run")
}

// TestHandleResult_RoutesProjectExecutionCorrectly tests that research_and_crawl routes to handleProjectResult
func TestHandleResult_RoutesProjectExecutionCorrectly(t *testing.T) {
	ctx := context.Background()

	mockProject := &mockProjectClientForRouting{}
	mockState := &mockStateUseCaseForRouting{
		getStateResult: &models.ProjectState{
			Status: models.ProjectStatusCrawling,
			Total:  100,
			Done:   1,
			Errors: 0,
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

	payload := createCrawlerContentForRouting("research_and_crawl", "proj123-brand-0", "tiktok")
	result := models.CrawlerResult{
		Success: true,
		Payload: payload,
	}

	err := uc.HandleResult(ctx, result)

	require.NoError(t, err)
	// Verify project execution flow was called
	assert.True(t, mockState.incrementDoneCalled, "IncrementDone should be called")
	assert.Equal(t, "proj123", mockState.incrementDoneProjectID)
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
	assert.False(t, mockState.incrementDoneCalled, "IncrementDone should NOT be called")
}

// TestExtractProjectID_WithBrandSuffix tests extracting project ID from job_id with -brand- suffix
func TestExtractProjectID_WithBrandSuffix(t *testing.T) {
	ctx := context.Background()
	uc := implUseCase{l: &mockLogger{}}

	payload := createCrawlerContentForRouting("research_and_crawl", "proj123-brand-0", "tiktok")

	projectID, err := uc.extractProjectID(ctx, payload)

	require.NoError(t, err)
	assert.Equal(t, "proj123", projectID)
}

// TestExtractProjectID_WithoutBrandSuffix tests extracting project ID when no -brand- suffix
func TestExtractProjectID_WithoutBrandSuffix(t *testing.T) {
	ctx := context.Background()
	uc := implUseCase{l: &mockLogger{}}

	payload := createCrawlerContentForRouting("research_and_crawl", "simple-job-id", "tiktok")

	projectID, err := uc.extractProjectID(ctx, payload)

	require.NoError(t, err)
	assert.Equal(t, "simple-job-id", projectID)
}

// TestExtractProjectID_ComplexProjectID tests extracting project ID with complex format
func TestExtractProjectID_ComplexProjectID(t *testing.T) {
	ctx := context.Background()
	uc := implUseCase{l: &mockLogger{}}

	// Project ID itself contains hyphens
	payload := createCrawlerContentForRouting("research_and_crawl", "proj-abc-123-brand-5", "tiktok")

	projectID, err := uc.extractProjectID(ctx, payload)

	require.NoError(t, err)
	assert.Equal(t, "proj-abc-123", projectID)
}

// TestHandleResult_ProjectExecutionWithErrors tests error counter increment
func TestHandleResult_ProjectExecutionWithErrors(t *testing.T) {
	ctx := context.Background()

	mockProject := &mockProjectClientForRouting{}
	mockState := &mockStateUseCaseForRouting{
		getStateResult: &models.ProjectState{
			Status: models.ProjectStatusCrawling,
			Total:  100,
			Done:   0,
			Errors: 1,
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

	payload := createCrawlerContentForRouting("research_and_crawl", "proj123-brand-0", "tiktok")
	result := models.CrawlerResult{
		Success: false, // Failed result
		Payload: payload,
	}

	err := uc.HandleResult(ctx, result)

	require.NoError(t, err)
	// Verify error counter was incremented
	assert.True(t, mockState.incrementErrorsCalled, "IncrementErrors should be called for failed result")
	assert.Equal(t, "proj123", mockState.incrementErrorsProjectID)
	assert.False(t, mockState.incrementDoneCalled, "IncrementDone should NOT be called for failed result")
}
