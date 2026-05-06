package rabbitmq

import (
	"testing"
	"time"

	"ingest-srv/internal/dryrun"

	"github.com/stretchr/testify/require"
)

func TestNewDispatchMessage(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	tcs := map[string]struct {
		input  dryrun.PublishDispatchInput
		mock   struct{}
		output DispatchMessage
		err    error
	}{
		"success": {
			input:  dryrun.PublishDispatchInput{TaskID: "task-1", Action: dryrun.ActionNameFullFlow, Params: map[string]interface{}{"limit": 1}, CreatedAt: now},
			output: DispatchMessage{TaskID: "task-1", Action: string(dryrun.ActionNameFullFlow), Params: map[string]interface{}{"limit": 1}, CreatedAt: now.Format(time.RFC3339)},
		},
	}
	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, tc.output, NewDispatchMessage(tc.input))
		})
	}
}

func TestMarshalDispatchMessage(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	tcs := map[string]struct {
		input  dryrun.PublishDispatchInput
		mock   struct{}
		output []byte
		err    error
	}{
		"success": {input: dryrun.PublishDispatchInput{TaskID: "task-1", Action: dryrun.ActionNameFullFlow, CreatedAt: now}},
		"marshal_error": {
			input: dryrun.PublishDispatchInput{TaskID: "task-1", Action: dryrun.ActionNameFullFlow, Params: map[string]interface{}{"bad": func() {}}, CreatedAt: now},
			err:   dryrun.ErrDispatchFailed,
		},
	}
	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			body, err := MarshalDispatchMessage(tc.input)
			if tc.err != nil {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Contains(t, string(body), `"task_id":"task-1"`)
		})
	}
}

func TestToHandleCompletionInput(t *testing.T) {
	count := 3
	tcs := map[string]struct {
		input  CompletionMessage
		mock   struct{}
		output dryrun.HandleCompletionInput
		err    error
	}{
		"success": {
			input: CompletionMessage{TaskID: "task-1", Status: "success", CompletedAt: "now", StorageBucket: "bucket", StoragePath: "path", BatchID: "batch", Checksum: "sum", ItemCount: &count, Error: "err", Metadata: map[string]interface{}{"k": "v"}},
			output: dryrun.HandleCompletionInput{
				TaskID: "task-1", Status: "success", CompletedAt: "now", StorageBucket: "bucket", StoragePath: "path", BatchID: "batch", Checksum: "sum", ItemCount: &count, Error: "err", Metadata: map[string]interface{}{"k": "v"},
			},
		},
	}
	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, tc.output, tc.input.ToHandleCompletionInput())
		})
	}
}
