package usecase

import (
	"encoding/json"
	"testing"
	"time"

	"ingest-srv/internal/execution"
	repo "ingest-srv/internal/execution/repository"
	"ingest-srv/internal/model"

	"github.com/smap-hcmut/shared-libs/go/log"
	"github.com/stretchr/testify/require"
)

func newTestUseCase() *implUseCase {
	return &implUseCase{l: log.NewZapLogger(log.ZapConfig{Level: log.LevelFatal, Mode: log.ModeProduction, Encoding: log.EncodingJSON})}
}

func TestValidateDispatchContext(t *testing.T) {
	uc := newTestUseCase()
	mode := model.CrawlModeNormal

	tcs := map[string]struct {
		input  repo.DispatchContext
		mock   struct{}
		output struct{}
		err    error
	}{
		"success": {
			input: repo.DispatchContext{Source: model.DataSource{SourceCategory: model.SourceCategoryCrawl, Status: model.SourceStatusActive, CrawlMode: &mode}, Target: model.CrawlTarget{IsActive: true}},
		},
		"passive": {
			input: repo.DispatchContext{Source: model.DataSource{SourceCategory: model.SourceCategoryPassive, Status: model.SourceStatusActive, CrawlMode: &mode}, Target: model.CrawlTarget{IsActive: true}},
			err:   execution.ErrDispatchNotAllowed,
		},
		"inactive_target": {
			input: repo.DispatchContext{Source: model.DataSource{SourceCategory: model.SourceCategoryCrawl, Status: model.SourceStatusActive, CrawlMode: &mode}, Target: model.CrawlTarget{}},
			err:   execution.ErrDispatchNotAllowed,
		},
		"missing_mode": {
			input: repo.DispatchContext{Source: model.DataSource{SourceCategory: model.SourceCategoryCrawl, Status: model.SourceStatusActive}, Target: model.CrawlTarget{IsActive: true}},
			err:   execution.ErrDispatchNotAllowed,
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			err := uc.validateDispatchContext(tc.input)

			if tc.err != nil {
				require.Equal(t, tc.err, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestBuildDispatchSpecs(t *testing.T) {
	uc := newTestUseCase()

	tcs := map[string]struct {
		input struct {
			source model.DataSource
			target model.CrawlTarget
		}
		mock   struct{}
		output int
		err    error
	}{
		"tiktok_keyword": {
			input: struct {
				source model.DataSource
				target model.CrawlTarget
			}{source: model.DataSource{SourceType: model.SourceTypeTikTok}, target: model.CrawlTarget{TargetType: model.TargetTypeKeyword, Values: []string{" vinfast ", "vf8"}}},
			output: 2,
		},
		"facebook_post_url": {
			input: struct {
				source model.DataSource
				target model.CrawlTarget
			}{source: model.DataSource{SourceType: model.SourceTypeFacebook}, target: model.CrawlTarget{TargetType: model.TargetTypePostURL, PlatformMeta: json.RawMessage(`{"parse_ids":["1"," 2 "]}`)}},
			output: 1,
		},
		"empty_keyword": {
			input: struct {
				source model.DataSource
				target model.CrawlTarget
			}{source: model.DataSource{SourceType: model.SourceTypeTikTok}, target: model.CrawlTarget{TargetType: model.TargetTypeKeyword, Values: []string{" "}}},
			err: execution.ErrDispatchNotAllowed,
		},
		"unsupported": {
			input: struct {
				source model.DataSource
				target model.CrawlTarget
			}{source: model.DataSource{SourceType: model.SourceTypeWebhook}, target: model.CrawlTarget{TargetType: model.TargetTypeKeyword}},
			err: execution.ErrUnsupportedDispatchMapping,
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			output, err := uc.buildDispatchSpecs(tc.input.source, tc.input.target)

			if tc.err != nil {
				require.Equal(t, tc.err, err)
				return
			}
			require.NoError(t, err)
			require.Len(t, output, tc.output)
		})
	}
}

func TestExecutionHelpers(t *testing.T) {
	uc := newTestUseCase()
	mode := model.CrawlModeNormal
	now := time.Date(2026, 5, 5, 1, 2, 3, 0, time.UTC)

	tcs := map[string]struct {
		input  struct{}
		mock   struct{}
		output struct{}
		err    error
	}{
		"success": {},
	}

	for name := range tcs {
		t.Run(name, func(t *testing.T) {
			duration, err := uc.computeEffectiveInterval(model.DataSource{CrawlMode: &mode}, model.CrawlTarget{CrawlIntervalMinutes: 10})
			require.NoError(t, err)
			require.Equal(t, 10*time.Minute, duration)
			require.Equal(t, "NORMAL", uc.derefCrawlMode(&mode))
			require.Equal(t, "", uc.derefCrawlMode(nil))
			require.Equal(t, now.Format(time.RFC3339), uc.formatTimePtr(&now))
			require.Equal(t, "", uc.formatTimePtr(nil))
			require.Equal(t, []string{"a", "b"}, uc.extractKeywords([]string{" a ", "", "b"}))
			require.True(t, uc.isTerminalFailure(model.JobStatusFailed))
			require.False(t, uc.isTerminalFailure(model.JobStatusSuccess))
		})
	}
}
