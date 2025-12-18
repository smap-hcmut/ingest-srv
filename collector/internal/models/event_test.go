package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// ============================================================================
// Phase 7: Completion Logic Tests
// Property 7: Completion Logic
// Validates: Requirements 3.4, 4.3, 4.4
// ============================================================================

// TestIsCrawlComplete_TaskLevel tests crawl completion using task-level counters
func TestIsCrawlComplete_TaskLevel(t *testing.T) {
	tests := []struct {
		name       string
		state      ProjectState
		wantResult bool
	}{
		{
			name: "complete when tasks_done equals tasks_total",
			state: ProjectState{
				TasksTotal:  10,
				TasksDone:   10,
				TasksErrors: 0,
			},
			wantResult: true,
		},
		{
			name: "complete when tasks_done + tasks_errors equals tasks_total",
			state: ProjectState{
				TasksTotal:  10,
				TasksDone:   8,
				TasksErrors: 2,
			},
			wantResult: true,
		},
		{
			name: "complete when tasks_done + tasks_errors exceeds tasks_total",
			state: ProjectState{
				TasksTotal:  10,
				TasksDone:   8,
				TasksErrors: 5,
			},
			wantResult: true,
		},
		{
			name: "not complete when tasks_done + tasks_errors less than tasks_total",
			state: ProjectState{
				TasksTotal:  10,
				TasksDone:   5,
				TasksErrors: 2,
			},
			wantResult: false,
		},
		{
			name: "not complete when tasks_total is zero",
			state: ProjectState{
				TasksTotal:  0,
				TasksDone:   5,
				TasksErrors: 0,
			},
			wantResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.state.IsCrawlComplete()
			assert.Equal(t, tt.wantResult, result)
		})
	}
}

// TestIsCrawlComplete_LegacyFallback tests crawl completion fallback to legacy fields
func TestIsCrawlComplete_LegacyFallback(t *testing.T) {
	tests := []struct {
		name       string
		state      ProjectState
		wantResult bool
	}{
		{
			name: "fallback to legacy when tasks_total is zero - complete",
			state: ProjectState{
				TasksTotal:  0,
				CrawlTotal:  10,
				CrawlDone:   10,
				CrawlErrors: 0,
			},
			wantResult: true,
		},
		{
			name: "fallback to legacy when tasks_total is zero - not complete",
			state: ProjectState{
				TasksTotal:  0,
				CrawlTotal:  10,
				CrawlDone:   5,
				CrawlErrors: 2,
			},
			wantResult: false,
		},
		{
			name: "prefer task-level over legacy when both available",
			state: ProjectState{
				TasksTotal:  10,
				TasksDone:   10,
				TasksErrors: 0,
				CrawlTotal:  100, // Different value
				CrawlDone:   50,  // Would be incomplete if used
				CrawlErrors: 0,
			},
			wantResult: true, // Uses task-level
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.state.IsCrawlComplete()
			assert.Equal(t, tt.wantResult, result)
		})
	}
}

// TestIsAnalyzeComplete tests analyze phase completion
func TestIsAnalyzeComplete(t *testing.T) {
	tests := []struct {
		name       string
		state      ProjectState
		wantResult bool
	}{
		{
			name: "complete when analyze_done equals analyze_total",
			state: ProjectState{
				AnalyzeTotal:  100,
				AnalyzeDone:   100,
				AnalyzeErrors: 0,
			},
			wantResult: true,
		},
		{
			name: "complete when analyze_done + analyze_errors equals analyze_total",
			state: ProjectState{
				AnalyzeTotal:  100,
				AnalyzeDone:   90,
				AnalyzeErrors: 10,
			},
			wantResult: true,
		},
		{
			name: "complete when analyze_done + analyze_errors exceeds analyze_total",
			state: ProjectState{
				AnalyzeTotal:  100,
				AnalyzeDone:   90,
				AnalyzeErrors: 20,
			},
			wantResult: true,
		},
		{
			name: "not complete when analyze_done + analyze_errors less than analyze_total",
			state: ProjectState{
				AnalyzeTotal:  100,
				AnalyzeDone:   50,
				AnalyzeErrors: 10,
			},
			wantResult: false,
		},
		{
			name: "not complete when analyze_total is zero",
			state: ProjectState{
				AnalyzeTotal:  0,
				AnalyzeDone:   50,
				AnalyzeErrors: 0,
			},
			wantResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.state.IsAnalyzeComplete()
			assert.Equal(t, tt.wantResult, result)
		})
	}
}

// TestIsComplete tests overall project completion (both phases)
func TestIsComplete(t *testing.T) {
	tests := []struct {
		name       string
		state      ProjectState
		wantResult bool
	}{
		{
			name: "complete when both crawl and analyze are complete",
			state: ProjectState{
				TasksTotal:    10,
				TasksDone:     10,
				TasksErrors:   0,
				AnalyzeTotal:  100,
				AnalyzeDone:   100,
				AnalyzeErrors: 0,
			},
			wantResult: true,
		},
		{
			name: "not complete when only crawl is complete",
			state: ProjectState{
				TasksTotal:    10,
				TasksDone:     10,
				TasksErrors:   0,
				AnalyzeTotal:  100,
				AnalyzeDone:   50,
				AnalyzeErrors: 0,
			},
			wantResult: false,
		},
		{
			name: "not complete when only analyze is complete",
			state: ProjectState{
				TasksTotal:    10,
				TasksDone:     5,
				TasksErrors:   0,
				AnalyzeTotal:  100,
				AnalyzeDone:   100,
				AnalyzeErrors: 0,
			},
			wantResult: false,
		},
		{
			name: "not complete when neither phase is complete",
			state: ProjectState{
				TasksTotal:    10,
				TasksDone:     5,
				TasksErrors:   0,
				AnalyzeTotal:  100,
				AnalyzeDone:   50,
				AnalyzeErrors: 0,
			},
			wantResult: false,
		},
		{
			name: "complete with errors in both phases",
			state: ProjectState{
				TasksTotal:    10,
				TasksDone:     8,
				TasksErrors:   2,
				AnalyzeTotal:  100,
				AnalyzeDone:   90,
				AnalyzeErrors: 10,
			},
			wantResult: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.state.IsComplete()
			assert.Equal(t, tt.wantResult, result)
		})
	}
}

// ============================================================================
// Phase 8: Legacy Field Consistency Tests
// Property 8: Legacy Field Consistency
// Validates: Requirements 5.2
// ============================================================================

// TestSetTasksTotal_SetsLegacyCrawlTotal tests that SetTasksTotal also sets crawl_total
// Note: This is tested at the state usecase level, not model level
// The model tests verify the completion logic uses both fields correctly

// TestCrawlProgressPercent_ItemLevelPreferred tests that item-level is preferred for progress
func TestCrawlProgressPercent_ItemLevelPreferred(t *testing.T) {
	tests := []struct {
		name       string
		state      ProjectState
		wantResult float64
	}{
		{
			name: "uses item-level when items_expected > 0",
			state: ProjectState{
				ItemsExpected: 100,
				ItemsActual:   50,
				ItemsErrors:   10,
				TasksTotal:    10, // Would give different result
				TasksDone:     5,
				TasksErrors:   0,
			},
			wantResult: 60.0, // (50+10)/100 * 100
		},
		{
			name: "falls back to task-level when items_expected is 0",
			state: ProjectState{
				ItemsExpected: 0,
				ItemsActual:   50,
				ItemsErrors:   10,
				TasksTotal:    10,
				TasksDone:     5,
				TasksErrors:   2,
			},
			wantResult: 70.0, // (5+2)/10 * 100
		},
		{
			name: "falls back to legacy crawl when both item and task are 0",
			state: ProjectState{
				ItemsExpected: 0,
				TasksTotal:    0,
				CrawlTotal:    100,
				CrawlDone:     80,
				CrawlErrors:   5,
			},
			wantResult: 85.0, // (80+5)/100 * 100
		},
		{
			name: "returns 0 when all totals are 0",
			state: ProjectState{
				ItemsExpected: 0,
				TasksTotal:    0,
				CrawlTotal:    0,
			},
			wantResult: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.state.CrawlProgressPercent()
			assert.Equal(t, tt.wantResult, result)
		})
	}
}

// TestTasksProgressPercent tests task-level progress calculation
func TestTasksProgressPercent(t *testing.T) {
	tests := []struct {
		name       string
		state      ProjectState
		wantResult float64
	}{
		{
			name: "calculates correct percentage",
			state: ProjectState{
				TasksTotal:  10,
				TasksDone:   5,
				TasksErrors: 2,
			},
			wantResult: 70.0, // (5+2)/10 * 100
		},
		{
			name: "returns 0 when tasks_total is 0",
			state: ProjectState{
				TasksTotal:  0,
				TasksDone:   5,
				TasksErrors: 2,
			},
			wantResult: 0.0,
		},
		{
			name: "returns 100 when all tasks complete",
			state: ProjectState{
				TasksTotal:  10,
				TasksDone:   10,
				TasksErrors: 0,
			},
			wantResult: 100.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.state.TasksProgressPercent()
			assert.Equal(t, tt.wantResult, result)
		})
	}
}

// TestItemsProgressPercent tests item-level progress calculation
func TestItemsProgressPercent(t *testing.T) {
	tests := []struct {
		name       string
		state      ProjectState
		wantResult float64
	}{
		{
			name: "calculates correct percentage",
			state: ProjectState{
				ItemsExpected: 100,
				ItemsActual:   50,
				ItemsErrors:   10,
			},
			wantResult: 60.0, // (50+10)/100 * 100
		},
		{
			name: "returns 0 when items_expected is 0",
			state: ProjectState{
				ItemsExpected: 0,
				ItemsActual:   50,
				ItemsErrors:   10,
			},
			wantResult: 0.0,
		},
		{
			name: "returns 100 when all items processed",
			state: ProjectState{
				ItemsExpected: 100,
				ItemsActual:   90,
				ItemsErrors:   10,
			},
			wantResult: 100.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.state.ItemsProgressPercent()
			assert.Equal(t, tt.wantResult, result)
		})
	}
}

// ============================================================================
// Phase 10: Progress Calculation Tests
// Property 9: Progress Calculation
// Validates: Requirements 6.2, 6.3
// ============================================================================

// TestAnalyzeProgressPercent tests analyze phase progress calculation
func TestAnalyzeProgressPercent(t *testing.T) {
	tests := []struct {
		name       string
		state      ProjectState
		wantResult float64
	}{
		{
			name: "calculates correct percentage",
			state: ProjectState{
				AnalyzeTotal:  100,
				AnalyzeDone:   50,
				AnalyzeErrors: 10,
			},
			wantResult: 60.0, // (50+10)/100 * 100
		},
		{
			name: "returns 0 when analyze_total is 0",
			state: ProjectState{
				AnalyzeTotal:  0,
				AnalyzeDone:   50,
				AnalyzeErrors: 10,
			},
			wantResult: 0.0,
		},
		{
			name: "returns 100 when all items analyzed",
			state: ProjectState{
				AnalyzeTotal:  100,
				AnalyzeDone:   90,
				AnalyzeErrors: 10,
			},
			wantResult: 100.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.state.AnalyzeProgressPercent()
			assert.Equal(t, tt.wantResult, result)
		})
	}
}

// TestOverallProgressPercent tests overall progress calculation
func TestOverallProgressPercent(t *testing.T) {
	tests := []struct {
		name       string
		state      ProjectState
		wantResult float64
	}{
		{
			name: "average of crawl and analyze progress",
			state: ProjectState{
				ItemsExpected: 100,
				ItemsActual:   80,
				ItemsErrors:   0,
				AnalyzeTotal:  80,
				AnalyzeDone:   40,
				AnalyzeErrors: 0,
			},
			wantResult: 65.0, // (80 + 50) / 2
		},
		{
			name: "both phases complete",
			state: ProjectState{
				ItemsExpected: 100,
				ItemsActual:   100,
				ItemsErrors:   0,
				AnalyzeTotal:  100,
				AnalyzeDone:   100,
				AnalyzeErrors: 0,
			},
			wantResult: 100.0,
		},
		{
			name: "both phases at 0",
			state: ProjectState{
				ItemsExpected: 0,
				TasksTotal:    0,
				CrawlTotal:    0,
				AnalyzeTotal:  0,
			},
			wantResult: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.state.OverallProgressPercent()
			assert.Equal(t, tt.wantResult, result)
		})
	}
}
