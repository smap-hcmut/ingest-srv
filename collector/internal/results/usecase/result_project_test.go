package usecase

import (
	"context"
	"testing"

	"smap-collector/internal/models"

	"github.com/stretchr/testify/assert"
)

// ============================================================================
// Tests for handleProjectResult with FLAT CrawlerResultMessage (Phase 4)
// ============================================================================

// TestHandleProjectResult_FlatFormat tests handleProjectResult with FLAT format v3.0
func TestHandleProjectResult_FlatFormat(t *testing.T) {
	ctx := context.Background()

	t.Run("success with valid FLAT message", func(t *testing.T) {
		// Setup mocks using existing mock types
		mockState := &mockStateUseCaseForRouting{
			getStateResult: &models.ProjectState{
				Status:        models.ProjectStatusProcessing,
				TasksTotal:    4,
				TasksDone:     1,
				ItemsExpected: 200,
				ItemsActual:   48,
				AnalyzeTotal:  48,
			},
			getUserIDResult: "user123",
		}
		mockWebhook := &mockWebhookUseCaseForRouting{}
		mockProject := &mockProjectClientForRouting{}

		uc := implUseCase{
			l:             &mockLogger{},
			stateUC:       mockState,
			webhookUC:     mockWebhook,
			projectClient: mockProject,
		}

		// Create FLAT format CrawlerResult directly (simulating RabbitMQ message)
		res := models.CrawlerResult{
			Success:         true,
			TaskType:        "research_and_crawl",
			JobID:           "proj123-brand-0",
			Platform:        "tiktok",
			RequestedLimit:  50,
			AppliedLimit:    50,
			TotalFound:      50,
			PlatformLimited: false,
			Successful:      48,
			Failed:          2,
			Skipped:         0,
		}

		// Execute
		err := uc.handleProjectResult(ctx, res)

		// Assert
		assert.NoError(t, err)
		assert.True(t, mockState.incrementCrawlDoneCalled, "IncrementTasksDone should be called")
		assert.Equal(t, "proj123", mockState.incrementCrawlDoneProjectID)
		assert.True(t, mockState.getStateCalled, "GetState should be called")
	})

	t.Run("platform limited - logs warning", func(t *testing.T) {
		mockState := &mockStateUseCaseForRouting{
			getStateResult: &models.ProjectState{
				Status: models.ProjectStatusProcessing,
			},
			getUserIDResult: "user123",
		}
		mockWebhook := &mockWebhookUseCaseForRouting{}
		mockProject := &mockProjectClientForRouting{}

		uc := implUseCase{
			l:             &mockLogger{},
			stateUC:       mockState,
			webhookUC:     mockWebhook,
			projectClient: mockProject,
		}

		// Create FLAT format CrawlerResult with platform limitation
		res := models.CrawlerResult{
			Success:         true,
			TaskType:        "research_and_crawl",
			JobID:           "proj123-brand-0",
			Platform:        "tiktok",
			RequestedLimit:  50,
			AppliedLimit:    50,
			TotalFound:      30,
			PlatformLimited: true,
			Successful:      28,
			Failed:          2,
			Skipped:         0,
		}

		err := uc.handleProjectResult(ctx, res)

		assert.NoError(t, err)
		// Platform limitation is logged but doesn't affect processing
		assert.True(t, mockState.incrementCrawlDoneCalled)
	})

	t.Run("failed task - increments errors", func(t *testing.T) {
		mockState := &mockStateUseCaseForRouting{
			getStateResult: &models.ProjectState{
				Status: models.ProjectStatusProcessing,
			},
			getUserIDResult: "user123",
		}
		mockWebhook := &mockWebhookUseCaseForRouting{}
		mockProject := &mockProjectClientForRouting{}

		uc := implUseCase{
			l:             &mockLogger{},
			stateUC:       mockState,
			webhookUC:     mockWebhook,
			projectClient: mockProject,
		}

		errCode := "SEARCH_FAILED"
		errMsg := "TikTok API rate limited"
		res := models.CrawlerResult{
			Success:         false,
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

		err := uc.handleProjectResult(ctx, res)

		assert.NoError(t, err)
		// Failed task should increment errors
		assert.True(t, mockState.incrementCrawlErrorsCalled, "IncrementTasksErrors should be called")
		assert.Equal(t, "proj123", mockState.incrementCrawlErrorsProjectID)
	})

	t.Run("project completion - sends completion webhook", func(t *testing.T) {
		mockState := &mockStateUseCaseForRouting{
			getStateResult: &models.ProjectState{
				Status:       models.ProjectStatusProcessing,
				TasksTotal:   4,
				TasksDone:    4,
				AnalyzeTotal: 200,
				AnalyzeDone:  200,
			},
			getUserIDResult:       "user123",
			checkCompletionResult: true,
		}
		mockWebhook := &mockWebhookUseCaseForRouting{}
		mockProject := &mockProjectClientForRouting{}

		uc := implUseCase{
			l:             &mockLogger{},
			stateUC:       mockState,
			webhookUC:     mockWebhook,
			projectClient: mockProject,
		}

		res := models.CrawlerResult{
			Success:         true,
			TaskType:        "research_and_crawl",
			JobID:           "proj123-brand-3",
			Platform:        "tiktok",
			RequestedLimit:  50,
			AppliedLimit:    50,
			TotalFound:      50,
			PlatformLimited: false,
			Successful:      50,
			Failed:          0,
			Skipped:         0,
		}

		err := uc.handleProjectResult(ctx, res)

		assert.NoError(t, err)
		assert.True(t, mockState.checkCompletionCalled, "CheckCompletion should be called")
		assert.True(t, mockWebhook.notifyCompletionCalled, "NotifyCompletion should be called on completion")
	})
}

// TestHandleProjectResult_Validation tests validation of FLAT message
func TestHandleProjectResult_Validation(t *testing.T) {
	ctx := context.Background()

	t.Run("invalid message - missing task_type", func(t *testing.T) {
		uc := implUseCase{l: &mockLogger{}}

		res := models.CrawlerResult{
			Success:  true,
			TaskType: "", // Missing
			JobID:    "proj123-brand-0",
			Platform: "tiktok",
		}

		err := uc.handleProjectResult(ctx, res)

		assert.Error(t, err)
	})

	t.Run("invalid message - missing job_id", func(t *testing.T) {
		uc := implUseCase{l: &mockLogger{}}

		res := models.CrawlerResult{
			Success:  true,
			TaskType: "research_and_crawl",
			JobID:    "", // Missing
			Platform: "tiktok",
		}

		err := uc.handleProjectResult(ctx, res)

		assert.Error(t, err)
	})

	t.Run("invalid message - missing platform", func(t *testing.T) {
		uc := implUseCase{l: &mockLogger{}}

		res := models.CrawlerResult{
			Success:  true,
			TaskType: "research_and_crawl",
			JobID:    "proj123-brand-0",
			Platform: "", // Missing
		}

		err := uc.handleProjectResult(ctx, res)

		assert.Error(t, err)
	})
}

// TestExtractProjectIDFromFlatMessage tests project ID extraction from FLAT message
func TestExtractProjectIDFromFlatMessage(t *testing.T) {
	tests := []struct {
		name       string
		jobID      string
		expectedID string
	}{
		{
			name:       "brand keyword format",
			jobID:      "proj123-brand-0",
			expectedID: "proj123",
		},
		{
			name:       "competitor keyword format",
			jobID:      "proj123-competitor-0",
			expectedID: "proj123",
		},
		{
			name:       "complex project ID with hyphens",
			jobID:      "proj-123-abc-brand-5",
			expectedID: "proj-123-abc",
		},
		{
			name:       "no suffix - returns full job_id",
			jobID:      "proj123",
			expectedID: "proj123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := &models.CrawlerResultMessage{
				JobID: tt.jobID,
			}

			projectID := msg.ExtractProjectID()
			assert.Equal(t, tt.expectedID, projectID)
		})
	}
}
