package models

import (
	"testing"
	"time"
)

func TestTransformProjectEventToRequests(t *testing.T) {
	tests := []struct {
		name          string
		event         ProjectCreatedEvent
		opts          TransformOptions
		expectedCount int
	}{
		{
			name: "brand keywords only",
			event: ProjectCreatedEvent{
				EventID:   "evt_123",
				Timestamp: time.Now(),
				Payload: ProjectCreatedPayload{
					ProjectID:     "proj_1",
					UserID:        "user_1",
					BrandName:     "TestBrand",
					BrandKeywords: []string{"keyword1", "keyword2"},
					DateRange:     DateRange{From: "2025-01-01", To: "2025-01-31"},
				},
			},
			opts:          DefaultTransformOptions(),
			expectedCount: 2,
		},
		{
			name: "brand and competitor keywords",
			event: ProjectCreatedEvent{
				EventID:   "evt_456",
				Timestamp: time.Now(),
				Payload: ProjectCreatedPayload{
					ProjectID:     "proj_2",
					UserID:        "user_2",
					BrandName:     "TestBrand",
					BrandKeywords: []string{"brand1"},
					CompetitorKeywordsMap: map[string][]string{
						"Competitor1": {"comp1_kw1", "comp1_kw2"},
						"Competitor2": {"comp2_kw1"},
					},
					DateRange: DateRange{From: "2025-01-01", To: "2025-02-01"},
				},
			},
			opts:          DefaultTransformOptions(),
			expectedCount: 4, // 1 brand + 2 comp1 + 1 comp2
		},
		{
			name: "empty keywords",
			event: ProjectCreatedEvent{
				EventID:   "evt_789",
				Timestamp: time.Now(),
				Payload: ProjectCreatedPayload{
					ProjectID: "proj_3",
					UserID:    "user_3",
					BrandName: "TestBrand",
				},
			},
			opts:          DefaultTransformOptions(),
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requests := TransformProjectEventToRequests(tt.event, tt.opts)
			if len(requests) != tt.expectedCount {
				t.Errorf("expected %d requests, got %d", tt.expectedCount, len(requests))
			}

			// Verify each request has required fields
			for _, req := range requests {
				if req.JobID == "" {
					t.Error("JobID should not be empty")
				}
				if req.TaskType != TaskTypeResearchAndCrawl {
					t.Errorf("expected TaskType %s, got %s", TaskTypeResearchAndCrawl, req.TaskType)
				}
				if req.Payload == nil {
					t.Error("Payload should not be nil")
				}
				if req.Payload["project_id"] != tt.event.Payload.ProjectID {
					t.Errorf("expected project_id %s, got %s", tt.event.Payload.ProjectID, req.Payload["project_id"])
				}
			}
		})
	}
}

func TestCalculateTimeRange(t *testing.T) {
	tests := []struct {
		name     string
		dr       DateRange
		expected int
	}{
		{
			name:     "valid 30 day range",
			dr:       DateRange{From: "2025-01-01", To: "2025-01-31"},
			expected: 30,
		},
		{
			name:     "valid 7 day range",
			dr:       DateRange{From: "2025-01-01", To: "2025-01-08"},
			expected: 7,
		},
		{
			name:     "empty from",
			dr:       DateRange{From: "", To: "2025-01-31"},
			expected: 30, // default
		},
		{
			name:     "empty to",
			dr:       DateRange{From: "2025-01-01", To: ""},
			expected: 30, // default
		},
		{
			name:     "invalid date format",
			dr:       DateRange{From: "01-01-2025", To: "31-01-2025"},
			expected: 30, // default
		},
		{
			name:     "negative range (to before from)",
			dr:       DateRange{From: "2025-01-31", To: "2025-01-01"},
			expected: 30, // default
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateTimeRange(tt.dr)
			if result != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, result)
			}
		})
	}
}

func TestCountTotalTasks(t *testing.T) {
	tests := []struct {
		name          string
		event         ProjectCreatedEvent
		platformCount int
		expected      int
	}{
		{
			name: "2 platforms, 3 keywords",
			event: ProjectCreatedEvent{
				Payload: ProjectCreatedPayload{
					BrandKeywords: []string{"kw1", "kw2", "kw3"},
				},
			},
			platformCount: 2,
			expected:      6, // 3 * 2
		},
		{
			name: "2 platforms, brand + competitor",
			event: ProjectCreatedEvent{
				Payload: ProjectCreatedPayload{
					BrandKeywords: []string{"brand1"},
					CompetitorKeywordsMap: map[string][]string{
						"Comp1": {"c1", "c2"},
					},
				},
			},
			platformCount: 2,
			expected:      6, // (1 + 2) * 2
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CountTotalTasks(tt.event, tt.platformCount)
			if result != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, result)
			}
		})
	}
}

func TestProjectCreatedEvent_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		event    ProjectCreatedEvent
		expected bool
	}{
		{
			name: "valid event",
			event: ProjectCreatedEvent{
				EventID: "evt_123",
				Payload: ProjectCreatedPayload{
					ProjectID: "proj_1",
					UserID:    "user_1",
				},
			},
			expected: true,
		},
		{
			name: "missing event_id",
			event: ProjectCreatedEvent{
				EventID: "",
				Payload: ProjectCreatedPayload{
					ProjectID: "proj_1",
					UserID:    "user_1",
				},
			},
			expected: false,
		},
		{
			name: "missing project_id",
			event: ProjectCreatedEvent{
				EventID: "evt_123",
				Payload: ProjectCreatedPayload{
					ProjectID: "",
					UserID:    "user_1",
				},
			},
			expected: false,
		},
		{
			name: "missing user_id",
			event: ProjectCreatedEvent{
				EventID: "evt_123",
				Payload: ProjectCreatedPayload{
					ProjectID: "proj_1",
					UserID:    "",
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.event.IsValid()
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// Note: DataCollectedEvent tests removed - DataCollectedEvent is published by Crawler services, not Collector.
// See document/event-drivent.md for the correct event flow.

func TestProjectStatus_IsTerminal(t *testing.T) {
	tests := []struct {
		status   ProjectStatus
		expected bool
	}{
		{ProjectStatusDone, true},
		{ProjectStatusFailed, true},
		{ProjectStatusInitializing, false},
		{ProjectStatusProcessing, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			if tt.status.IsTerminal() != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, tt.status.IsTerminal())
			}
		})
	}
}

func TestProjectState_IsCrawlComplete(t *testing.T) {
	tests := []struct {
		name     string
		state    ProjectState
		expected bool
	}{
		{
			name:     "complete - all done",
			state:    ProjectState{CrawlTotal: 10, CrawlDone: 10, CrawlErrors: 0},
			expected: true,
		},
		{
			name:     "complete - done + errors >= total",
			state:    ProjectState{CrawlTotal: 10, CrawlDone: 8, CrawlErrors: 2},
			expected: true,
		},
		{
			name:     "incomplete",
			state:    ProjectState{CrawlTotal: 10, CrawlDone: 5, CrawlErrors: 1},
			expected: false,
		},
		{
			name:     "zero total",
			state:    ProjectState{CrawlTotal: 0, CrawlDone: 0, CrawlErrors: 0},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.state.IsCrawlComplete() != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, tt.state.IsCrawlComplete())
			}
		})
	}
}

func TestProjectState_IsAnalyzeComplete(t *testing.T) {
	tests := []struct {
		name     string
		state    ProjectState
		expected bool
	}{
		{
			name:     "complete - all done",
			state:    ProjectState{AnalyzeTotal: 10, AnalyzeDone: 10, AnalyzeErrors: 0},
			expected: true,
		},
		{
			name:     "complete - done + errors >= total",
			state:    ProjectState{AnalyzeTotal: 10, AnalyzeDone: 8, AnalyzeErrors: 2},
			expected: true,
		},
		{
			name:     "incomplete",
			state:    ProjectState{AnalyzeTotal: 10, AnalyzeDone: 5, AnalyzeErrors: 1},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.state.IsAnalyzeComplete() != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, tt.state.IsAnalyzeComplete())
			}
		})
	}
}

func TestProjectState_IsComplete(t *testing.T) {
	tests := []struct {
		name     string
		state    ProjectState
		expected bool
	}{
		{
			name: "complete - both phases done",
			state: ProjectState{
				CrawlTotal: 10, CrawlDone: 10, CrawlErrors: 0,
				AnalyzeTotal: 10, AnalyzeDone: 10, AnalyzeErrors: 0,
			},
			expected: true,
		},
		{
			name: "incomplete - crawl done but analyze not",
			state: ProjectState{
				CrawlTotal: 10, CrawlDone: 10, CrawlErrors: 0,
				AnalyzeTotal: 10, AnalyzeDone: 5, AnalyzeErrors: 0,
			},
			expected: false,
		},
		{
			name: "incomplete - neither done",
			state: ProjectState{
				CrawlTotal: 10, CrawlDone: 5, CrawlErrors: 0,
				AnalyzeTotal: 5, AnalyzeDone: 2, AnalyzeErrors: 0,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.state.IsComplete() != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, tt.state.IsComplete())
			}
		})
	}
}

func TestProjectState_ProgressPercent(t *testing.T) {
	tests := []struct {
		name            string
		state           ProjectState
		expectedCrawl   float64
		expectedAnalyze float64
		expectedOverall float64
	}{
		{
			name: "50% crawl, 25% analyze",
			state: ProjectState{
				CrawlTotal: 100, CrawlDone: 50, CrawlErrors: 0,
				AnalyzeTotal: 100, AnalyzeDone: 25, AnalyzeErrors: 0,
			},
			expectedCrawl:   50.0,
			expectedAnalyze: 25.0,
			expectedOverall: 37.5,
		},
		{
			name: "zero totals",
			state: ProjectState{
				CrawlTotal: 0, CrawlDone: 0, CrawlErrors: 0,
				AnalyzeTotal: 0, AnalyzeDone: 0, AnalyzeErrors: 0,
			},
			expectedCrawl:   0.0,
			expectedAnalyze: 0.0,
			expectedOverall: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.state.CrawlProgressPercent() != tt.expectedCrawl {
				t.Errorf("crawl: expected %v, got %v", tt.expectedCrawl, tt.state.CrawlProgressPercent())
			}
			if tt.state.AnalyzeProgressPercent() != tt.expectedAnalyze {
				t.Errorf("analyze: expected %v, got %v", tt.expectedAnalyze, tt.state.AnalyzeProgressPercent())
			}
			if tt.state.OverallProgressPercent() != tt.expectedOverall {
				t.Errorf("overall: expected %v, got %v", tt.expectedOverall, tt.state.OverallProgressPercent())
			}
		})
	}
}
