package http

import (
	"errors"
	"testing"
	"time"

	"ingest-srv/internal/dryrun"
	"ingest-srv/internal/model"

	"github.com/stretchr/testify/require"
)

func TestMapError(t *testing.T) {
	h, _ := newTestHandler(t)

	tcs := map[string]struct {
		input  error
		mock   struct{}
		output error
		err    error
	}{
		"source_not_found": {input: dryrun.ErrSourceNotFound, output: errSourceNotFound},
		"target_not_found": {input: dryrun.ErrTargetNotFound, output: errTargetNotFound},
		"target_required":  {input: dryrun.ErrTargetRequired, output: errTargetRequired},
		"target_forbidden": {input: dryrun.ErrTargetForbidden, output: errTargetForbidden},
		"not_allowed":      {input: dryrun.ErrDryrunNotAllowed, output: errDryrunNotAllowed},
		"running":          {input: dryrun.ErrDryrunAlreadyRunning, output: errDryrunAlreadyRunning},
		"unsupported":      {input: dryrun.ErrUnsupportedMapping, output: errUnsupportedMapping},
		"invalid_limit":    {input: dryrun.ErrInvalidSampleLimit, output: errInvalidSampleLimit},
		"result_not_found": {input: dryrun.ErrResultNotFound, output: errResultNotFound},
		"dispatch_failed":  {input: dryrun.ErrDispatchFailed, output: errDispatchFailed},
		"create_failed":    {input: dryrun.ErrCreateFailed, output: errCreateFailed},
		"get_failed":       {input: dryrun.ErrGetFailed, output: errGetFailed},
		"update_failed":    {input: dryrun.ErrUpdateFailed, output: errUpdateFailed},
		"list_failed":      {input: dryrun.ErrListFailed, output: errListFailed},
		"default":          {input: errors.New("unknown"), output: errors.New("unknown")},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			output := h.mapError(tc.input)

			require.Equal(t, tc.output.Error(), output.Error())
		})
	}
}

func TestDryrunRequestValidate(t *testing.T) {
	limit := 5
	badLimit := 0

	tcs := map[string]struct {
		input  func() error
		mock   struct{}
		output struct{}
		err    error
	}{
		"trigger_success": {input: func() error {
			return triggerReq{SourceID: testSourceID, TargetID: testTargetID, SampleLimit: &limit}.validate()
		}},
		"trigger_bad_source": {input: func() error { return triggerReq{SourceID: "bad"}.validate() }, err: errWrongBody},
		"trigger_bad_target": {input: func() error { return triggerReq{SourceID: testSourceID, TargetID: "bad"}.validate() }, err: errWrongBody},
		"trigger_bad_limit":  {input: func() error { return triggerReq{SourceID: testSourceID, SampleLimit: &badLimit}.validate() }, err: errInvalidSampleLimit},
		"latest_success":     {input: func() error { return latestReq{SourceID: testSourceID, TargetID: testTargetID}.validate() }},
		"latest_bad_target":  {input: func() error { return latestReq{SourceID: testSourceID, TargetID: "bad"}.validate() }, err: errWrongBody},
		"history_success":    {input: func() error { return historyReq{SourceID: testSourceID, TargetID: testTargetID}.validate() }},
		"history_bad_source": {input: func() error { return historyReq{SourceID: "bad"}.validate() }, err: errWrongBody},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			err := tc.input()

			if tc.err != nil {
				require.Equal(t, tc.err, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestDryrunMappers(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	mode := model.CrawlModeNormal
	total := 10

	tcs := map[string]struct {
		input struct {
			result model.DryrunResult
			source model.DataSource
		}
		mock   struct{}
		output struct{}
		err    error
	}{
		"success": {
			input: struct {
				result model.DryrunResult
				source model.DataSource
			}{
				result: model.DryrunResult{ID: "result-1", SourceID: testSourceID, ProjectID: "project-1", TargetID: testTargetID, JobID: "job-1", Status: model.DryrunStatusSuccess, SampleCount: 1, TotalFound: &total, RequestedBy: "user-1", StartedAt: &now, CompletedAt: &now, CreatedAt: now},
				source: model.DataSource{ID: testSourceID, Status: model.SourceStatusActive, DryrunStatus: model.DryrunStatusSuccess, DryrunLastResultID: "result-1", CrawlMode: &mode},
			},
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			result := toDryrunResultResp(tc.input.result)
			source := toDataSourceSnapshotResp(tc.input.source)

			require.Equal(t, "result-1", result.ID)
			require.Equal(t, testSourceID, source.ID)
			require.Equal(t, "NORMAL", *source.CrawlMode)
			require.NotNil(t, result.StartedAt)
			require.Equal(t, now.Format(timeFormat), *result.StartedAt)
		})
	}
}
