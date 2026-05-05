package usecase

import (
	"encoding/json"
	"testing"
	"time"

	"ingest-srv/internal/dryrun"
	"ingest-srv/internal/model"

	"github.com/stretchr/testify/require"
)

func TestValidateTriggerInput(t *testing.T) {
	uc := &implUseCase{}
	zero := 0
	limit := 5

	tcs := map[string]struct {
		input  dryrun.TriggerInput
		mock   struct{}
		output struct{}
		err    error
	}{
		"success":         {input: dryrun.TriggerInput{SourceID: "source-1", SampleLimit: &limit}},
		"source_required": {input: dryrun.TriggerInput{}, err: dryrun.ErrSourceNotFound},
		"invalid_limit":   {input: dryrun.TriggerInput{SourceID: "source-1", SampleLimit: &zero}, err: dryrun.ErrInvalidSampleLimit},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			err := uc.validateTriggerInput(tc.input)

			if tc.err != nil {
				require.Equal(t, tc.err, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestValidateCompletionInput(t *testing.T) {
	uc := &implUseCase{}

	tcs := map[string]struct {
		input  dryrun.HandleCompletionInput
		mock   struct{}
		output struct{}
		err    error
	}{
		"success":          {input: dryrun.HandleCompletionInput{TaskID: "task-1", Status: "success", StorageBucket: "bucket", StoragePath: "path"}},
		"error_status":     {input: dryrun.HandleCompletionInput{TaskID: "task-1", Status: "error"}},
		"task_required":    {input: dryrun.HandleCompletionInput{}, err: dryrun.ErrInvalidCompletionInput},
		"missing_artifact": {input: dryrun.HandleCompletionInput{TaskID: "task-1", Status: "success"}, err: dryrun.ErrInvalidCompletionInput},
		"bad_status":       {input: dryrun.HandleCompletionInput{TaskID: "task-1", Status: "bad"}, err: dryrun.ErrInvalidCompletionInput},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			err := uc.validateCompletionInput(tc.input)

			if tc.err != nil {
				require.Equal(t, tc.err, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestDryrunHelpers(t *testing.T) {
	uc := &implUseCase{}
	limit := 7
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
			require.Equal(t, model.DryrunSampleLimitDefault, uc.normalizeSampleLimit(nil))
			require.Equal(t, limit, uc.normalizeSampleLimit(&limit))
			require.Equal(t, &now, uc.parseCompletedAt(now.Format(time.RFC3339)))
			require.Nil(t, uc.parseCompletedAt("bad"))
			require.True(t, uc.isTerminalDryrunStatus(model.DryrunStatusSuccess))
			require.Equal(t, []string{"a", "b"}, uc.extractKeywords([]string{" a ", "", "b"}))
			require.Equal(t, map[string]interface{}{"keyword": "vinfast", "limit": 7, "threshold": 0.5, "comment_count": 50, dryrun.ParamKeyRuntimeKind: string(dryrun.RuntimeKindDryrun)}, uc.buildKeywordSampleParams("vinfast", 7))
		})
	}
}

func TestBuildSamplePayload(t *testing.T) {
	uc := &implUseCase{}
	fallback := 9

	tcs := map[string]struct {
		input struct {
			raw               json.RawMessage
			limit             int
			artifactItemCount int
			fallback          *int
		}
		mock   struct{}
		output struct {
			count int
			total *int
		}
		err error
	}{
		"array_truncated": {
			input: struct {
				raw                      json.RawMessage
				limit, artifactItemCount int
				fallback                 *int
			}{raw: json.RawMessage(`[1,2,3]`), limit: 2},
			output: struct {
				count int
				total *int
			}{count: 2, total: intPtr(3)},
		},
		"object_items": {
			input: struct {
				raw                      json.RawMessage
				limit, artifactItemCount int
				fallback                 *int
			}{raw: json.RawMessage(`{"items":[{"id":1}],"total_posts":8}`), limit: 5},
			output: struct {
				count int
				total *int
			}{count: 1, total: intPtr(8)},
		},
		"null": {
			input: struct {
				raw                      json.RawMessage
				limit, artifactItemCount int
				fallback                 *int
			}{raw: json.RawMessage(`null`), limit: 5, fallback: &fallback},
			output: struct {
				count int
				total *int
			}{count: 0, total: &fallback},
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			_, count, total, _ := uc.buildSamplePayload(tc.input.raw, tc.input.limit, tc.input.artifactItemCount, tc.input.fallback)

			require.Equal(t, tc.output.count, count)
			require.Equal(t, tc.output.total, total)
		})
	}
}

func intPtr(v int) *int {
	return &v
}
