package http

import (
	"errors"
	"testing"

	"ingest-srv/internal/execution"

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
		"datasource_not_found": {input: execution.ErrDataSourceNotFound, output: errDatasourceNotFound},
		"target_not_found":     {input: execution.ErrTargetNotFound, output: errTargetNotFound},
		"not_allowed":          {input: execution.ErrDispatchNotAllowed, output: errDispatchNotAllowed},
		"unsupported":          {input: execution.ErrUnsupportedDispatchMapping, output: errUnsupportedMapping},
		"parse_ids":            {input: execution.ErrPlatformMetaParseIDs, output: errParseIDsRequired},
		"default":              {input: errors.New("unknown"), output: errDispatchFailed},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			output := h.mapError(tc.input)

			require.Equal(t, tc.output, output)
		})
	}
}

func TestDispatchReqValidate(t *testing.T) {
	tcs := map[string]struct {
		input  dispatchReq
		mock   struct{}
		output execution.DispatchTargetManuallyInput
		err    error
	}{
		"success": {
			input:  dispatchReq{DataSourceID: " " + testSourceID + " ", TargetID: testTargetID},
			output: execution.DispatchTargetManuallyInput{DataSourceID: testSourceID, TargetID: testTargetID},
		},
		"bad_source": {input: dispatchReq{DataSourceID: "bad", TargetID: testTargetID}, err: errWrongPath},
		"bad_target": {input: dispatchReq{DataSourceID: testSourceID, TargetID: "bad"}, err: errWrongPath},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			err := tc.input.validate()

			if tc.err != nil {
				require.Equal(t, tc.err, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.output, tc.input.toInput())
		})
	}
}
