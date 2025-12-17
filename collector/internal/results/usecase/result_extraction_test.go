package usecase

import (
	"context"
	"testing"

	"smap-collector/internal/models"

	"github.com/stretchr/testify/assert"
)

// ============================================================================
// Tests for extractLimitInfoAndStats (Phase 8.3.1)
// ============================================================================

func TestExtractLimitInfoAndStats_EnhancedResponse(t *testing.T) {
	ctx := context.Background()
	uc := implUseCase{l: &mockLogger{}}

	t.Run("with enhanced response - limit_info and stats present", func(t *testing.T) {
		// Create enhanced response with limit_info and stats
		enhancedPayload := map[string]any{
			"success": true,
			"limit_info": map[string]any{
				"requested_limit":  50,
				"applied_limit":    50,
				"total_found":      45,
				"platform_limited": false,
			},
			"stats": map[string]any{
				"successful":      45,
				"failed":          0,
				"skipped":         5,
				"completion_rate": 0.9,
			},
			"payload": []any{},
		}

		res := models.CrawlerResult{
			Success: true,
			Payload: enhancedPayload,
		}

		limitInfo, stats := uc.extractLimitInfoAndStats(ctx, res)

		// Should use fallback since the payload structure doesn't match EnhancedCrawlerResult directly
		assert.NotNil(t, limitInfo)
		assert.NotNil(t, stats)
	})

	t.Run("with old response - fallback", func(t *testing.T) {
		// Old format: just success and payload array
		oldPayload := []map[string]any{
			{
				"meta": map[string]any{
					"id":        "video1",
					"platform":  "youtube",
					"job_id":    "proj_1-brand-0",
					"task_type": "research_and_crawl",
				},
				"content": map[string]any{
					"text": "Test video",
				},
			},
			{
				"meta": map[string]any{
					"id":        "video2",
					"platform":  "youtube",
					"job_id":    "proj_1-brand-0",
					"task_type": "research_and_crawl",
				},
				"content": map[string]any{
					"text": "Test video 2",
				},
			},
		}

		res := models.CrawlerResult{
			Success: true,
			Payload: oldPayload,
		}

		limitInfo, stats := uc.extractLimitInfoAndStats(ctx, res)

		assert.NotNil(t, limitInfo)
		assert.NotNil(t, stats)
		// Fallback should count items
		assert.Equal(t, 2, stats.Successful)
		assert.Equal(t, 0, stats.Failed)
	})

	t.Run("with empty payload", func(t *testing.T) {
		res := models.CrawlerResult{
			Success: true,
			Payload: []any{},
		}

		limitInfo, stats := uc.extractLimitInfoAndStats(ctx, res)

		assert.NotNil(t, limitInfo)
		assert.NotNil(t, stats)
		// Empty payload should default to 1
		assert.Equal(t, 1, stats.Successful)
	})

	t.Run("with nil payload", func(t *testing.T) {
		res := models.CrawlerResult{
			Success: true,
			Payload: nil,
		}

		limitInfo, stats := uc.extractLimitInfoAndStats(ctx, res)

		assert.NotNil(t, limitInfo)
		assert.NotNil(t, stats)
		// Nil payload should default to 1
		assert.Equal(t, 1, stats.Successful)
	})

	t.Run("with failed response", func(t *testing.T) {
		res := models.CrawlerResult{
			Success: false,
			Payload: nil,
		}

		limitInfo, stats := uc.extractLimitInfoAndStats(ctx, res)

		assert.NotNil(t, limitInfo)
		assert.NotNil(t, stats)
		// Failed response should count as failed
		assert.Equal(t, 0, stats.Successful)
		assert.Equal(t, 1, stats.Failed)
	})
}

func TestFallbackLimitInfoAndStats(t *testing.T) {
	ctx := context.Background()
	uc := implUseCase{l: &mockLogger{}}

	t.Run("success with multiple items", func(t *testing.T) {
		payload := []map[string]any{
			{"meta": map[string]any{"id": "1"}},
			{"meta": map[string]any{"id": "2"}},
			{"meta": map[string]any{"id": "3"}},
		}

		res := models.CrawlerResult{
			Success: true,
			Payload: payload,
		}

		limitInfo, stats := uc.fallbackLimitInfoAndStats(ctx, res)

		assert.NotNil(t, limitInfo)
		assert.Equal(t, 50, limitInfo.RequestedLimit) // default
		assert.Equal(t, 50, limitInfo.AppliedLimit)
		assert.Equal(t, 3, limitInfo.TotalFound)
		assert.False(t, limitInfo.PlatformLimited)

		assert.NotNil(t, stats)
		assert.Equal(t, 3, stats.Successful)
		assert.Equal(t, 0, stats.Failed)
		assert.Equal(t, 0, stats.Skipped)
		assert.Equal(t, 1.0, stats.CompletionRate)
	})

	t.Run("failure with items", func(t *testing.T) {
		payload := []map[string]any{
			{"meta": map[string]any{"id": "1"}},
			{"meta": map[string]any{"id": "2"}},
		}

		res := models.CrawlerResult{
			Success: false,
			Payload: payload,
		}

		_, stats := uc.fallbackLimitInfoAndStats(ctx, res)

		assert.NotNil(t, stats)
		assert.Equal(t, 0, stats.Successful)
		assert.Equal(t, 2, stats.Failed)
	})

	t.Run("empty payload defaults to 1", func(t *testing.T) {
		res := models.CrawlerResult{
			Success: true,
			Payload: []any{},
		}

		limitInfo, stats := uc.fallbackLimitInfoAndStats(ctx, res)

		assert.NotNil(t, limitInfo)
		assert.Equal(t, 1, limitInfo.TotalFound)
		assert.Equal(t, 1, stats.Successful)
	})
}

// ============================================================================
// Tests for countBatchItems (Phase 8.3.1)
// ============================================================================

func TestCountBatchItems(t *testing.T) {
	ctx := context.Background()
	uc := implUseCase{l: &mockLogger{}}

	t.Run("count multiple items", func(t *testing.T) {
		payload := []map[string]any{
			{"meta": map[string]any{"id": "1", "platform": "youtube", "job_id": "job1"}},
			{"meta": map[string]any{"id": "2", "platform": "youtube", "job_id": "job1"}},
			{"meta": map[string]any{"id": "3", "platform": "youtube", "job_id": "job1"}},
		}

		count := uc.countBatchItems(ctx, payload)
		assert.Equal(t, 3, count)
	})

	t.Run("empty payload", func(t *testing.T) {
		count := uc.countBatchItems(ctx, []any{})
		assert.Equal(t, 0, count)
	})

	t.Run("nil payload", func(t *testing.T) {
		count := uc.countBatchItems(ctx, nil)
		assert.Equal(t, 0, count)
	})

	t.Run("invalid payload type", func(t *testing.T) {
		count := uc.countBatchItems(ctx, "invalid")
		assert.Equal(t, 0, count)
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
// Tests for CrawlError.IsRetryable (Phase 8.3.2)
// ============================================================================

func TestCrawlError_IsRetryable(t *testing.T) {
	tests := []struct {
		name     string
		err      *models.CrawlError
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "AUTH_FAILED - not retryable",
			err:      &models.CrawlError{Code: "AUTH_FAILED", Message: "Authentication failed"},
			expected: false,
		},
		{
			name:     "INVALID_KEYWORD - not retryable",
			err:      &models.CrawlError{Code: "INVALID_KEYWORD", Message: "Invalid keyword"},
			expected: false,
		},
		{
			name:     "BLOCKED - not retryable",
			err:      &models.CrawlError{Code: "BLOCKED", Message: "Account blocked"},
			expected: false,
		},
		{
			name:     "RATE_LIMITED_PERMANENT - not retryable",
			err:      &models.CrawlError{Code: "RATE_LIMITED_PERMANENT", Message: "Permanent rate limit"},
			expected: false,
		},
		{
			name:     "TIMEOUT - retryable",
			err:      &models.CrawlError{Code: "TIMEOUT", Message: "Request timeout"},
			expected: true,
		},
		{
			name:     "NETWORK_ERROR - retryable",
			err:      &models.CrawlError{Code: "NETWORK_ERROR", Message: "Network error"},
			expected: true,
		},
		{
			name:     "RATE_LIMITED - retryable",
			err:      &models.CrawlError{Code: "RATE_LIMITED", Message: "Temporary rate limit"},
			expected: true,
		},
		{
			name:     "UNKNOWN - retryable",
			err:      &models.CrawlError{Code: "UNKNOWN", Message: "Unknown error"},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.err.IsRetryable()
			assert.Equal(t, tt.expected, result)
		})
	}
}
