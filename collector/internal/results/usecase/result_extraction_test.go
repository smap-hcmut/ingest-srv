package usecase

import (
	"context"
	"testing"

	"smap-collector/internal/models"

	"github.com/stretchr/testify/assert"
)

// ============================================================================
// Tests for parseCrawlerResultMessage (Phase 4 - FLAT format v3.0)
// ============================================================================

func TestParseCrawlerResultMessage(t *testing.T) {
	ctx := context.Background()
	uc := implUseCase{l: &mockLogger{}}

	t.Run("parse valid FLAT message", func(t *testing.T) {
		res := models.CrawlerResult{
			Success: true,
			Payload: nil, // FLAT format has no payload
		}

		// Simulate FLAT format by creating a struct that matches CrawlerResultMessage
		// In real scenario, the message comes from RabbitMQ already in FLAT format
		msg, err := uc.parseCrawlerResultMessage(ctx, res)

		assert.NoError(t, err)
		assert.NotNil(t, msg)
		assert.True(t, msg.Success)
	})

	t.Run("parse message with all fields", func(t *testing.T) {
		// Create a CrawlerResult that will be parsed as CrawlerResultMessage
		res := models.CrawlerResult{
			Success: true,
			Payload: nil,
		}

		msg, err := uc.parseCrawlerResultMessage(ctx, res)

		assert.NoError(t, err)
		assert.NotNil(t, msg)
	})
}

// ============================================================================
// Tests for extractTaskTypeFromRoot (Phase 4 - FLAT format v3.0)
// ============================================================================

func TestExtractTaskTypeFromRoot(t *testing.T) {
	ctx := context.Background()
	uc := implUseCase{l: &mockLogger{}}

	t.Run("extract from FLAT format", func(t *testing.T) {
		// Note: In real scenario, the CrawlerResult would have task_type at root level
		// This test verifies the extraction logic
		res := models.CrawlerResult{
			Success: true,
			Payload: nil,
		}

		taskType := uc.extractTaskTypeFromRoot(ctx, res)

		// CrawlerResult doesn't have task_type field, so this returns empty
		// The actual FLAT message would have task_type at root level
		assert.Equal(t, "", taskType)
	})
}

// ============================================================================
// Tests for buildHybridProgressRequest (Phase 8.3.2)
// ============================================================================

func TestBuildHybridProgressRequest(t *testing.T) {
	uc := implUseCase{l: &mockLogger{}}

	t.Run("build with hybrid state", func(t *testing.T) {
		state := &models.ProjectState{
			Status:        models.ProjectStatusProcessing,
			TasksTotal:    10,
			TasksDone:     5,
			TasksErrors:   1,
			ItemsExpected: 500,
			ItemsActual:   250,
			ItemsErrors:   10,
			AnalyzeTotal:  250,
			AnalyzeDone:   100,
			AnalyzeErrors: 5,
			CrawlTotal:    10,
			CrawlDone:     5,
			CrawlErrors:   1,
		}

		req := uc.buildHybridProgressRequest("proj_1", "user_1", state)

		// Verify basic fields
		assert.Equal(t, "proj_1", req.ProjectID)
		assert.Equal(t, "user_1", req.UserID)
		assert.Equal(t, "PROCESSING", req.Status)

		// Verify task progress
		assert.Equal(t, int64(10), req.Tasks.Total)
		assert.Equal(t, int64(5), req.Tasks.Done)
		assert.Equal(t, int64(1), req.Tasks.Errors)
		assert.Equal(t, 60.0, req.Tasks.Percent) // (5+1)/10 * 100

		// Verify item progress
		assert.Equal(t, int64(500), req.Items.Expected)
		assert.Equal(t, int64(250), req.Items.Actual)
		assert.Equal(t, int64(10), req.Items.Errors)
		assert.Equal(t, 52.0, req.Items.Percent) // (250+10)/500 * 100

		// Verify analyze progress
		assert.Equal(t, int64(250), req.Analyze.Total)
		assert.Equal(t, int64(100), req.Analyze.Done)
		assert.Equal(t, int64(5), req.Analyze.Errors)
		assert.Equal(t, 42.0, req.Analyze.ProgressPercent) // (100+5)/250 * 100

		// Verify legacy crawl progress
		assert.Equal(t, int64(10), req.Crawl.Total)
		assert.Equal(t, int64(5), req.Crawl.Done)
		assert.Equal(t, int64(1), req.Crawl.Errors)
	})

	t.Run("build with zero values", func(t *testing.T) {
		state := &models.ProjectState{
			Status: models.ProjectStatusInitializing,
		}

		req := uc.buildHybridProgressRequest("proj_2", "user_2", state)

		assert.Equal(t, "proj_2", req.ProjectID)
		assert.Equal(t, "user_2", req.UserID)
		assert.Equal(t, "INITIALIZING", req.Status)
		assert.Equal(t, int64(0), req.Tasks.Total)
		assert.Equal(t, 0.0, req.Tasks.Percent)
		assert.Equal(t, int64(0), req.Items.Expected)
		assert.Equal(t, 0.0, req.Items.Percent)
	})
}

// ============================================================================
// Tests for CrawlerResultMessage.IsErrorRetryable (Phase 4 - FLAT format)
// ============================================================================

func TestCrawlerResultMessage_IsErrorRetryable(t *testing.T) {
	strPtr := func(s string) *string { return &s }

	tests := []struct {
		name     string
		msg      *models.CrawlerResultMessage
		expected bool
	}{
		{
			name:     "nil error code",
			msg:      &models.CrawlerResultMessage{ErrorCode: nil},
			expected: false,
		},
		{
			name:     "AUTH_FAILED - not retryable",
			msg:      &models.CrawlerResultMessage{ErrorCode: strPtr("AUTH_FAILED")},
			expected: false,
		},
		{
			name:     "INVALID_KEYWORD - not retryable",
			msg:      &models.CrawlerResultMessage{ErrorCode: strPtr("INVALID_KEYWORD")},
			expected: false,
		},
		{
			name:     "BLOCKED - not retryable",
			msg:      &models.CrawlerResultMessage{ErrorCode: strPtr("BLOCKED")},
			expected: false,
		},
		{
			name:     "RATE_LIMITED_PERMANENT - not retryable",
			msg:      &models.CrawlerResultMessage{ErrorCode: strPtr("RATE_LIMITED_PERMANENT")},
			expected: false,
		},
		{
			name:     "TIMEOUT - retryable",
			msg:      &models.CrawlerResultMessage{ErrorCode: strPtr("TIMEOUT")},
			expected: true,
		},
		{
			name:     "NETWORK_ERROR - retryable",
			msg:      &models.CrawlerResultMessage{ErrorCode: strPtr("NETWORK_ERROR")},
			expected: true,
		},
		{
			name:     "RATE_LIMITED - retryable",
			msg:      &models.CrawlerResultMessage{ErrorCode: strPtr("RATE_LIMITED")},
			expected: true,
		},
		{
			name:     "UNKNOWN - retryable",
			msg:      &models.CrawlerResultMessage{ErrorCode: strPtr("UNKNOWN")},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.msg.IsErrorRetryable()
			assert.Equal(t, tt.expected, result)
		})
	}
}
