package models

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// CrawlerResultMessage Tests (FLAT FORMAT v3.0)
// **Feature: project-state-contract-fix, Property 1: Message Structure Validation**
// **Validates: Requirements 1.1, 1.4**
// ============================================================================

// TestCrawlerResultMessageValidation_Valid tests that valid messages pass validation
func TestCrawlerResultMessageValidation_Valid(t *testing.T) {
	msg := CrawlerResultMessage{
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

	err := msg.Validate()
	assert.NoError(t, err)
}

// TestCrawlerResultMessageValidation_MissingFields tests validation for missing required fields
func TestCrawlerResultMessageValidation_MissingFields(t *testing.T) {
	testCases := []struct {
		name string
		msg  CrawlerResultMessage
		err  error
	}{
		{
			name: "missing task_type",
			msg:  CrawlerResultMessage{JobID: "job1", Platform: "tiktok"},
			err:  ErrMissingTaskType,
		},
		{
			name: "missing job_id",
			msg:  CrawlerResultMessage{TaskType: "research_and_crawl", Platform: "tiktok"},
			err:  ErrMissingJobID,
		},
		{
			name: "missing platform",
			msg:  CrawlerResultMessage{TaskType: "research_and_crawl", JobID: "job1"},
			err:  ErrMissingPlatform,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.msg.Validate()
			assert.Error(t, err)
			assert.Equal(t, tc.err, err)
		})
	}
}

// TestCrawlerResultMessageValidation_NegativeValues tests validation for negative values
func TestCrawlerResultMessageValidation_NegativeValues(t *testing.T) {
	baseMsg := CrawlerResultMessage{
		TaskType: "research_and_crawl",
		JobID:    "proj123-brand-0",
		Platform: "tiktok",
	}

	t.Run("negative successful", func(t *testing.T) {
		msg := baseMsg
		msg.Successful = -1
		assert.Equal(t, ErrInvalidCounts, msg.Validate())
	})

	t.Run("negative failed", func(t *testing.T) {
		msg := baseMsg
		msg.Failed = -1
		assert.Equal(t, ErrInvalidCounts, msg.Validate())
	})

	t.Run("negative skipped", func(t *testing.T) {
		msg := baseMsg
		msg.Skipped = -1
		assert.Equal(t, ErrInvalidCounts, msg.Validate())
	})

	t.Run("negative requested_limit", func(t *testing.T) {
		msg := baseMsg
		msg.RequestedLimit = -1
		assert.Equal(t, ErrInvalidLimits, msg.Validate())
	})

	t.Run("negative applied_limit", func(t *testing.T) {
		msg := baseMsg
		msg.AppliedLimit = -1
		assert.Equal(t, ErrInvalidLimits, msg.Validate())
	})

	t.Run("negative total_found", func(t *testing.T) {
		msg := baseMsg
		msg.TotalFound = -1
		assert.Equal(t, ErrInvalidLimits, msg.Validate())
	})
}

// TestCrawlerResultMessageExtractProjectID tests project ID extraction from job_id
func TestCrawlerResultMessageExtractProjectID(t *testing.T) {
	testCases := []struct {
		jobID     string
		projectID string
	}{
		{"proj123-brand-0", "proj123"},
		{"proj123-brand-1", "proj123"},
		{"proj123-competitor-0", "proj123"},
		{"my-project-123-brand-0", "my-project-123"},
		{"simple-job-id", "simple-job-id"},
		{"proj-with-dashes-brand-5", "proj-with-dashes"},
	}

	for _, tc := range testCases {
		t.Run(tc.jobID, func(t *testing.T) {
			msg := CrawlerResultMessage{JobID: tc.jobID}
			assert.Equal(t, tc.projectID, msg.ExtractProjectID())
		})
	}
}

// TestCrawlerResultMessageUnmarshal tests JSON unmarshaling of flat format
func TestCrawlerResultMessageUnmarshal(t *testing.T) {
	jsonData := `{
		"success": true,
		"task_type": "research_and_crawl",
		"job_id": "proj123-brand-0",
		"platform": "tiktok",
		"requested_limit": 50,
		"applied_limit": 50,
		"total_found": 30,
		"platform_limited": true,
		"successful": 28,
		"failed": 2,
		"skipped": 0
	}`

	var msg CrawlerResultMessage
	err := json.Unmarshal([]byte(jsonData), &msg)

	require.NoError(t, err)
	assert.True(t, msg.Success)
	assert.Equal(t, "research_and_crawl", msg.TaskType)
	assert.Equal(t, "proj123-brand-0", msg.JobID)
	assert.Equal(t, "tiktok", msg.Platform)
	assert.Equal(t, 50, msg.RequestedLimit)
	assert.Equal(t, 50, msg.AppliedLimit)
	assert.Equal(t, 30, msg.TotalFound)
	assert.True(t, msg.PlatformLimited)
	assert.Equal(t, 28, msg.Successful)
	assert.Equal(t, 2, msg.Failed)
	assert.Equal(t, 0, msg.Skipped)
	assert.NoError(t, msg.Validate())
}

// TestCrawlerResultMessageUnmarshal_WithError tests JSON unmarshaling with error fields
func TestCrawlerResultMessageUnmarshal_WithError(t *testing.T) {
	jsonData := `{
		"success": false,
		"task_type": "research_and_crawl",
		"job_id": "proj123-brand-0",
		"platform": "tiktok",
		"requested_limit": 50,
		"applied_limit": 50,
		"total_found": 0,
		"platform_limited": false,
		"successful": 0,
		"failed": 0,
		"skipped": 0,
		"error_code": "SEARCH_FAILED",
		"error_message": "TikTok API rate limited"
	}`

	var msg CrawlerResultMessage
	err := json.Unmarshal([]byte(jsonData), &msg)

	require.NoError(t, err)
	assert.False(t, msg.Success)
	assert.NotNil(t, msg.ErrorCode)
	assert.Equal(t, "SEARCH_FAILED", *msg.ErrorCode)
	assert.NotNil(t, msg.ErrorMessage)
	assert.Equal(t, "TikTok API rate limited", *msg.ErrorMessage)
}

// TestCrawlerResultMessageIsErrorRetryable tests error retryability logic
func TestCrawlerResultMessageIsErrorRetryable(t *testing.T) {
	testCases := []struct {
		errorCode string
		retryable bool
	}{
		{"SEARCH_FAILED", true},
		{"RATE_LIMITED", true},
		{"TIMEOUT", true},
		{"AUTH_FAILED", false},
		{"INVALID_KEYWORD", false},
		{"BLOCKED", false},
		{"RATE_LIMITED_PERMANENT", false},
	}

	for _, tc := range testCases {
		t.Run(tc.errorCode, func(t *testing.T) {
			code := tc.errorCode
			msg := CrawlerResultMessage{ErrorCode: &code}
			assert.Equal(t, tc.retryable, msg.IsErrorRetryable())
		})
	}

	t.Run("nil error code", func(t *testing.T) {
		msg := CrawlerResultMessage{ErrorCode: nil}
		assert.False(t, msg.IsErrorRetryable())
	})
}
