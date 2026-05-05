package usecase

import (
	"encoding/json"
	"testing"

	"ingest-srv/internal/datasource"
	"ingest-srv/internal/model"

	"github.com/stretchr/testify/require"
)

func TestValidateSourceType(t *testing.T) {
	uc := &implUseCase{}

	tcs := map[string]struct {
		input  string
		mock   struct{}
		output struct{}
		err    error
	}{
		"tiktok":      {input: string(model.SourceTypeTikTok)},
		"facebook":    {input: string(model.SourceTypeFacebook)},
		"youtube":     {input: string(model.SourceTypeYouTube)},
		"file_upload": {input: string(model.SourceTypeFileUpload)},
		"webhook":     {input: string(model.SourceTypeWebhook)},
		"invalid":     {input: "bad", err: datasource.ErrInvalidSourceType},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			err := uc.validateSourceType(tc.input)

			if tc.err != nil {
				require.Equal(t, tc.err, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestPrepareTargetValues(t *testing.T) {
	uc := &implUseCase{}

	tcs := map[string]struct {
		input struct {
			targetType model.TargetType
			values     []string
		}
		mock   struct{}
		output []string
		err    error
	}{
		"keyword_trim_and_dedupe": {
			input: struct {
				targetType model.TargetType
				values     []string
			}{targetType: model.TargetTypeKeyword, values: []string{" pin ", "pin", "xe"}},
			output: []string{"pin", "xe"},
		},
		"profile_url": {
			input: struct {
				targetType model.TargetType
				values     []string
			}{targetType: model.TargetTypeProfile, values: []string{" https://example.com/a "}},
			output: []string{"https://example.com/a"},
		},
		"empty": {
			input: struct {
				targetType model.TargetType
				values     []string
			}{targetType: model.TargetTypeKeyword, values: []string{" "}},
			err: datasource.ErrTargetValuesRequired,
		},
		"bad_url": {
			input: struct {
				targetType model.TargetType
				values     []string
			}{targetType: model.TargetTypePostURL, values: []string{"not-url"}},
			err: datasource.ErrTargetValuesMustBeURLs,
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			output, err := uc.prepareTargetValues(tc.input.targetType, tc.input.values)

			if tc.err != nil {
				require.Equal(t, tc.err, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.output, output)
		})
	}
}

func TestAreJSONRawEqual(t *testing.T) {
	uc := &implUseCase{}

	tcs := map[string]struct {
		input struct {
			left  json.RawMessage
			right json.RawMessage
		}
		mock   struct{}
		output bool
		err    error
	}{
		"both_empty": {output: true},
		"semantic_equal": {
			input:  struct{ left, right json.RawMessage }{left: json.RawMessage(`{"a":1,"b":2}`), right: json.RawMessage(`{"b":2,"a":1}`)},
			output: true,
		},
		"trimmed_bytes_equal": {
			input:  struct{ left, right json.RawMessage }{left: json.RawMessage(` raw `), right: json.RawMessage(`raw`)},
			output: true,
		},
		"different": {
			input:  struct{ left, right json.RawMessage }{left: json.RawMessage(`{"a":1}`), right: json.RawMessage(`{"a":2}`)},
			output: false,
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			output := uc.areJSONRawEqual(tc.input.left, tc.input.right)

			require.Equal(t, tc.output, output)
		})
	}
}

func TestBuildProjectLifecycleUpdateOptions(t *testing.T) {
	uc := &implUseCase{}

	tcs := map[string]struct {
		input  string
		mock   struct{}
		output model.SourceStatus
		err    error
	}{
		"activate": {input: "activate", output: model.SourceStatusActive},
		"pause":    {input: "pause", output: model.SourceStatusPaused},
		"resume":   {input: "resume", output: model.SourceStatusActive},
		"default":  {input: "bad", output: ""},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			output := uc.buildProjectLifecycleUpdateOptions("project-1", tc.input)

			require.Equal(t, "project-1", output.ProjectID)
			require.Equal(t, tc.output, output.ToStatus)
		})
	}
}
