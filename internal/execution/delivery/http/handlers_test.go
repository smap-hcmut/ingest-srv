package http

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"ingest-srv/internal/execution"

	"github.com/gin-gonic/gin"
	"github.com/smap-hcmut/shared-libs/go/log"
	"github.com/stretchr/testify/require"
)

const (
	testSourceID = "550e8400-e29b-41d4-a716-446655440000"
	testTargetID = "550e8400-e29b-41d4-a716-446655440001"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func newTestHandler(t *testing.T) (*handler, *execution.MockUseCase) {
	t.Helper()
	l := log.NewZapLogger(log.ZapConfig{Level: log.LevelFatal, Mode: log.ModeProduction, Encoding: log.EncodingJSON})
	uc := execution.NewMockUseCase(t)
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

func TestDispatchTarget(t *testing.T) {
	type mockDispatch struct {
		isCalled bool
		input    execution.DispatchTargetManuallyInput
		output   execution.DispatchTargetManuallyOutput
		err      error
	}
	type mockData struct {
		dispatch mockDispatch
	}

	tcs := map[string]struct {
		input  gin.Params
		mock   mockData
		output int
		err    error
	}{
		"success": {
			input: gin.Params{{Key: "id", Value: testSourceID}, {Key: "target_id", Value: testTargetID}},
			mock: mockData{dispatch: mockDispatch{
				isCalled: true,
				input:    execution.DispatchTargetManuallyInput{DataSourceID: testSourceID, TargetID: testTargetID},
				output:   execution.DispatchTargetManuallyOutput{ScheduledJobID: "job-1", Status: "DISPATCHED", TaskCount: 1, PublishedCount: 1, Tasks: []execution.DispatchTaskOutput{{ExternalTaskID: "external-1", TaskID: "task-1", Queue: "queue", Action: "full_flow", Status: "PUBLISHED", Keyword: "vinfast"}}},
			}},
			output: http.StatusOK,
		},
		"wrong_path": {
			input:  gin.Params{{Key: "id", Value: "bad"}, {Key: "target_id", Value: testTargetID}},
			output: http.StatusBadRequest,
		},
		"uc_error": {
			input: gin.Params{{Key: "id", Value: testSourceID}, {Key: "target_id", Value: testTargetID}},
			mock: mockData{dispatch: mockDispatch{
				isCalled: true,
				input:    execution.DispatchTargetManuallyInput{DataSourceID: testSourceID, TargetID: testTargetID},
				err:      execution.ErrTargetNotFound,
			}},
			output: http.StatusNotFound,
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			h, uc := newTestHandler(t)
			if tc.mock.dispatch.isCalled {
				uc.EXPECT().DispatchTargetManually(context.Background(), tc.mock.dispatch.input).Return(tc.mock.dispatch.output, tc.mock.dispatch.err)
			}
			c, w := newTestContext(http.MethodPost, "/internal/datasources/"+testSourceID+"/targets/"+testTargetID+"/dispatch", "", tc.input)

			h.DispatchTarget(c)

			require.Equal(t, tc.output, w.Code)
		})
	}
}
