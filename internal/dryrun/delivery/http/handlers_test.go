package http

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"ingest-srv/internal/dryrun"
	"ingest-srv/internal/model"

	"github.com/gin-gonic/gin"
	"github.com/smap-hcmut/shared-libs/go/log"
	"github.com/smap-hcmut/shared-libs/go/paginator"
	"github.com/stretchr/testify/require"
)

const (
	testSourceID = "550e8400-e29b-41d4-a716-446655440000"
	testTargetID = "550e8400-e29b-41d4-a716-446655440001"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func newTestHandler(t *testing.T) (*handler, *dryrun.MockUseCase) {
	t.Helper()
	l := log.NewZapLogger(log.ZapConfig{Level: log.LevelFatal, Mode: log.ModeProduction, Encoding: log.EncodingJSON})
	uc := dryrun.NewMockUseCase(t)
	return New(l, uc, nil).(*handler), uc
}

func newTestContext(method, target, body string, params gin.Params) (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(method, target, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req
	c.Params = params
	return c, w
}

func TestTrigger(t *testing.T) {
	limit := 5
	type mockTrigger struct {
		isCalled bool
		input    dryrun.TriggerInput
		output   dryrun.TriggerOutput
		err      error
	}
	type mockData struct {
		trigger mockTrigger
	}

	tcs := map[string]struct {
		input  string
		mock   mockData
		output int
		err    error
	}{
		"success": {
			input: `{"target_id":"` + testTargetID + `","sample_limit":5,"force":true}`,
			mock: mockData{trigger: mockTrigger{
				isCalled: true,
				input:    dryrun.TriggerInput{SourceID: testSourceID, TargetID: testTargetID, SampleLimit: &limit, Force: true},
				output:   dryrun.TriggerOutput{Result: model.DryrunResult{ID: "result-1", CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}, DataSource: model.DataSource{ID: testSourceID}},
			}},
			output: http.StatusOK,
		},
		"wrong_body": {
			input:  `{`,
			output: http.StatusBadRequest,
		},
		"wrong_body_validate": {
			input:  `{"target_id":"bad"}`,
			output: http.StatusBadRequest,
		},
		"uc_error": {
			input: `{"target_id":"` + testTargetID + `"}`,
			mock: mockData{trigger: mockTrigger{
				isCalled: true,
				input:    dryrun.TriggerInput{SourceID: testSourceID, TargetID: testTargetID},
				err:      dryrun.ErrDryrunAlreadyRunning,
			}},
			output: http.StatusBadRequest,
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			h, uc := newTestHandler(t)
			if tc.mock.trigger.isCalled {
				uc.EXPECT().Trigger(context.Background(), tc.mock.trigger.input).Return(tc.mock.trigger.output, tc.mock.trigger.err)
			}
			c, w := newTestContext(http.MethodPost, "/datasources/"+testSourceID+"/dryrun", tc.input, gin.Params{{Key: "id", Value: testSourceID}})

			h.Trigger(c)

			require.Equal(t, tc.output, w.Code)
		})
	}
}

func TestGetLatest(t *testing.T) {
	type mockLatest struct {
		isCalled bool
		input    dryrun.GetLatestInput
		output   dryrun.GetLatestOutput
		err      error
	}
	type mockData struct {
		latest mockLatest
	}

	tcs := map[string]struct {
		input  string
		mock   mockData
		output int
		err    error
	}{
		"success": {
			input: "?target_id=" + testTargetID,
			mock: mockData{latest: mockLatest{
				isCalled: true,
				input:    dryrun.GetLatestInput{SourceID: testSourceID, TargetID: testTargetID},
				output:   dryrun.GetLatestOutput{Result: model.DryrunResult{ID: "result-1"}},
			}},
			output: http.StatusOK,
		},
		"wrong_query": {
			input:  "?target_id=bad",
			output: http.StatusBadRequest,
		},
		"wrong_path": {
			input:  "?target_id=" + testTargetID,
			output: http.StatusBadRequest,
		},
		"uc_error": {
			mock: mockData{latest: mockLatest{
				isCalled: true,
				input:    dryrun.GetLatestInput{SourceID: testSourceID},
				err:      dryrun.ErrResultNotFound,
			}},
			output: http.StatusNotFound,
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			h, uc := newTestHandler(t)
			if tc.mock.latest.isCalled {
				uc.EXPECT().GetLatest(context.Background(), tc.mock.latest.input).Return(tc.mock.latest.output, tc.mock.latest.err)
			}
			sourceID := testSourceID
			if name == "wrong_path" {
				sourceID = "bad"
			}
			c, w := newTestContext(http.MethodGet, "/datasources/"+sourceID+"/dryrun/latest"+tc.input, "", gin.Params{{Key: "id", Value: sourceID}})

			h.GetLatest(c)

			require.Equal(t, tc.output, w.Code)
		})
	}
}

func TestListHistory(t *testing.T) {
	type mockHistory struct {
		isCalled bool
		input    dryrun.ListHistoryInput
		output   dryrun.ListHistoryOutput
		err      error
	}
	type mockData struct {
		history mockHistory
	}

	tcs := map[string]struct {
		input  string
		mock   mockData
		output int
		err    error
	}{
		"success": {
			input: "?target_id=" + testTargetID + "&page=1&limit=10",
			mock: mockData{history: mockHistory{
				isCalled: true,
				input:    dryrun.ListHistoryInput{SourceID: testSourceID, TargetID: testTargetID, Paginator: paginator.PaginateQuery{Page: 1, Limit: 10}},
				output:   dryrun.ListHistoryOutput{Results: []model.DryrunResult{{ID: "result-1"}}, Paginator: paginator.Paginator{Total: 1}},
			}},
			output: http.StatusOK,
		},
		"wrong_query": {
			input:  "?target_id=bad",
			output: http.StatusBadRequest,
		},
		"wrong_query_bind": {
			input:  "?page=bad",
			output: http.StatusBadRequest,
		},
		"uc_error": {
			mock: mockData{history: mockHistory{
				isCalled: true,
				input:    dryrun.ListHistoryInput{SourceID: testSourceID},
				err:      dryrun.ErrListFailed,
			}},
			output: http.StatusInternalServerError,
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			h, uc := newTestHandler(t)
			if tc.mock.history.isCalled {
				uc.EXPECT().ListHistory(context.Background(), tc.mock.history.input).Return(tc.mock.history.output, tc.mock.history.err)
			}
			c, w := newTestContext(http.MethodGet, "/datasources/"+testSourceID+"/dryrun/history"+tc.input, "", gin.Params{{Key: "id", Value: testSourceID}})

			h.ListHistory(c)

			require.Equal(t, tc.output, w.Code)
		})
	}
}
