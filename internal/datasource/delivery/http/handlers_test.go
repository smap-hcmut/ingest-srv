package http

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"ingest-srv/internal/datasource"
	"ingest-srv/internal/model"

	"github.com/gin-gonic/gin"
	"github.com/smap-hcmut/shared-libs/go/log"
	"github.com/smap-hcmut/shared-libs/go/paginator"
	"github.com/stretchr/testify/require"
)

const (
	testSourceID  = "550e8400-e29b-41d4-a716-446655440000"
	testProjectID = "550e8400-e29b-41d4-a716-446655440001"
	testTargetID  = "550e8400-e29b-41d4-a716-446655440002"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func newTestHandler(t *testing.T) (*handler, *datasource.MockUseCase) {
	t.Helper()
	l := log.NewZapLogger(log.ZapConfig{Level: log.LevelFatal, Mode: log.ModeProduction, Encoding: log.EncodingJSON})
	uc := datasource.NewMockUseCase(t)
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

func testDataSource() model.DataSource {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	mode := model.CrawlModeNormal
	interval := 10
	return model.DataSource{
		ID:                   testSourceID,
		ProjectID:            testProjectID,
		Name:                 "TikTok source",
		Description:          "description",
		SourceType:           model.SourceTypeTikTok,
		SourceCategory:       model.SourceCategoryCrawl,
		Status:               model.SourceStatusPending,
		Config:               []byte(`{"k":"v"}`),
		AccountRef:           []byte(`{"account":"ref"}`),
		MappingRules:         []byte(`{"rule":"x"}`),
		OnboardingStatus:     model.OnboardingStatusNotRequired,
		DryrunStatus:         model.DryrunStatusPending,
		DryrunLastResultID:   "result-1",
		CrawlMode:            &mode,
		CrawlIntervalMinutes: &interval,
		NextCrawlAt:          &now,
		LastCrawlAt:          &now,
		LastSuccessAt:        &now,
		LastErrorAt:          &now,
		LastErrorMessage:     "last error",
		WebhookID:            "webhook-1",
		CreatedBy:            "user-1",
		ActivatedAt:          &now,
		PausedAt:             &now,
		ArchivedAt:           &now,
		CreatedAt:            now,
		UpdatedAt:            now,
	}
}

func testTarget() model.CrawlTarget {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	return model.CrawlTarget{
		ID:                   testTargetID,
		DataSourceID:         testSourceID,
		TargetType:           model.TargetTypeKeyword,
		Values:               []string{"vinfast"},
		Label:                "label",
		PlatformMeta:         []byte(`{"platform":"tiktok"}`),
		IsActive:             true,
		Priority:             1,
		CrawlIntervalMinutes: 10,
		NextCrawlAt:          &now,
		LastCrawlAt:          &now,
		LastSuccessAt:        &now,
		LastErrorAt:          &now,
		LastErrorMessage:     "target error",
		CreatedAt:            now,
		UpdatedAt:            now,
	}
}

func TestCreate(t *testing.T) {
	type mockCreate struct {
		isCalled bool
		input    datasource.CreateInput
		output   datasource.CreateOutput
		err      error
	}
	type mockData struct {
		create mockCreate
	}

	tcs := map[string]struct {
		input  string
		mock   mockData
		output int
		err    error
	}{
		"success": {
			input: `{"project_id":"` + testProjectID + `","name":" TikTok source ","source_type":"TIKTOK","config":{"k":"v"},"account_ref":{"account":"ref"},"mapping_rules":{"rule":"x"},"crawl_mode":"NORMAL","crawl_interval_minutes":10}`,
			mock: mockData{create: mockCreate{
				isCalled: true,
				input:    datasource.CreateInput{ProjectID: testProjectID, Name: "TikTok source", SourceType: string(model.SourceTypeTikTok), SourceCategory: string(model.SourceCategoryCrawl), Config: []byte(`{"k":"v"}`), AccountRef: []byte(`{"account":"ref"}`), MappingRules: []byte(`{"rule":"x"}`), CrawlMode: string(model.CrawlModeNormal), CrawlIntervalMinutes: 10},
				output:   datasource.CreateOutput{DataSource: testDataSource()},
			}},
			output: http.StatusOK,
		},
		"wrong_body": {
			input:  `{`,
			output: http.StatusBadRequest,
		},
		"wrong_body_validate": {
			input:  `{"project_id":"` + testProjectID + `","name":"TikTok source","source_type":"TIKTOK"}`,
			output: http.StatusBadRequest,
		},
		"uc_error": {
			input: `{"project_id":"` + testProjectID + `","name":"TikTok source","source_type":"TIKTOK","crawl_mode":"NORMAL","crawl_interval_minutes":10}`,
			mock: mockData{create: mockCreate{
				isCalled: true,
				input:    datasource.CreateInput{ProjectID: testProjectID, Name: "TikTok source", SourceType: string(model.SourceTypeTikTok), SourceCategory: string(model.SourceCategoryCrawl), CrawlMode: string(model.CrawlModeNormal), CrawlIntervalMinutes: 10},
				err:      datasource.ErrCreateFailed,
			}},
			output: http.StatusInternalServerError,
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			h, uc := newTestHandler(t)
			if tc.mock.create.isCalled {
				uc.EXPECT().Create(context.Background(), tc.mock.create.input).Return(tc.mock.create.output, tc.mock.create.err)
			}
			c, w := newTestContext(http.MethodPost, "/datasources", tc.input, nil)

			h.Create(c)

			require.Equal(t, tc.output, w.Code)
		})
	}
}

func TestDetail(t *testing.T) {
	type mockDetail struct {
		isCalled bool
		input    string
		output   datasource.DetailOutput
		err      error
	}
	type mockData struct {
		detail mockDetail
	}

	tcs := map[string]struct {
		input  gin.Params
		mock   mockData
		output int
		err    error
	}{
		"success": {
			input:  gin.Params{{Key: "id", Value: testSourceID}},
			mock:   mockData{detail: mockDetail{isCalled: true, input: testSourceID, output: datasource.DetailOutput{DataSource: testDataSource()}}},
			output: http.StatusOK,
		},
		"wrong_path": {
			input:  gin.Params{{Key: "id", Value: "bad"}},
			output: http.StatusBadRequest,
		},
		"uc_error": {
			input:  gin.Params{{Key: "id", Value: testSourceID}},
			mock:   mockData{detail: mockDetail{isCalled: true, input: testSourceID, err: datasource.ErrNotFound}},
			output: http.StatusNotFound,
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			h, uc := newTestHandler(t)
			if tc.mock.detail.isCalled {
				uc.EXPECT().Detail(context.Background(), tc.mock.detail.input).Return(tc.mock.detail.output, tc.mock.detail.err)
			}
			c, w := newTestContext(http.MethodGet, "/datasources/"+testSourceID, "", tc.input)

			h.Detail(c)

			require.Equal(t, tc.output, w.Code)
		})
	}
}

func TestList(t *testing.T) {
	type mockList struct {
		isCalled bool
		input    datasource.ListInput
		output   datasource.ListOutput
		err      error
	}
	type mockData struct {
		list mockList
	}

	tcs := map[string]struct {
		input  string
		mock   mockData
		output int
		err    error
	}{
		"success": {
			input: "?project_id=" + testProjectID + "&source_type=TIKTOK&source_category=CRAWL&crawl_mode=NORMAL&name=vinfast&page=1&limit=10",
			mock: mockData{list: mockList{
				isCalled: true,
				input:    datasource.ListInput{ProjectID: testProjectID, SourceType: string(model.SourceTypeTikTok), SourceCategory: string(model.SourceCategoryCrawl), CrawlMode: string(model.CrawlModeNormal), Name: "vinfast", Paginator: paginator.PaginateQuery{Page: 1, Limit: 10}},
				output:   datasource.ListOutput{DataSources: []model.DataSource{testDataSource()}, Paginator: paginator.Paginator{Total: 1}},
			}},
			output: http.StatusOK,
		},
		"wrong_query_bind": {
			input:  "?page=bad",
			output: http.StatusBadRequest,
		},
		"wrong_query_validate": {
			input:  "?source_type=BAD",
			output: http.StatusBadRequest,
		},
		"uc_error": {
			mock:   mockData{list: mockList{isCalled: true, err: datasource.ErrListFailed}},
			output: http.StatusInternalServerError,
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			h, uc := newTestHandler(t)
			if tc.mock.list.isCalled {
				uc.EXPECT().List(context.Background(), tc.mock.list.input).Return(tc.mock.list.output, tc.mock.list.err)
			}
			c, w := newTestContext(http.MethodGet, "/datasources"+tc.input, "", nil)

			h.List(c)

			require.Equal(t, tc.output, w.Code)
		})
	}
}

func TestUpdate(t *testing.T) {
	type mockUpdate struct {
		isCalled bool
		input    datasource.UpdateInput
		output   datasource.UpdateOutput
		err      error
	}
	type mockData struct {
		update mockUpdate
	}

	tcs := map[string]struct {
		input  string
		params gin.Params
		mock   mockData
		output int
		err    error
	}{
		"success": {
			input:  `{"name":" Updated ","description":"description","config":{"k":"v"}}`,
			params: gin.Params{{Key: "id", Value: testSourceID}},
			mock:   mockData{update: mockUpdate{isCalled: true, input: datasource.UpdateInput{ID: testSourceID, Name: "Updated", Description: "description", Config: []byte(`{"k":"v"}`)}, output: datasource.UpdateOutput{DataSource: testDataSource()}}},
			output: http.StatusOK,
		},
		"wrong_body": {
			input:  `{`,
			params: gin.Params{{Key: "id", Value: testSourceID}},
			output: http.StatusBadRequest,
		},
		"wrong_path": {
			input:  `{}`,
			params: gin.Params{{Key: "id", Value: "bad"}},
			output: http.StatusBadRequest,
		},
		"uc_error": {
			input:  `{}`,
			params: gin.Params{{Key: "id", Value: testSourceID}},
			mock:   mockData{update: mockUpdate{isCalled: true, input: datasource.UpdateInput{ID: testSourceID}, err: datasource.ErrUpdateFailed}},
			output: http.StatusInternalServerError,
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			h, uc := newTestHandler(t)
			if tc.mock.update.isCalled {
				uc.EXPECT().Update(context.Background(), tc.mock.update.input).Return(tc.mock.update.output, tc.mock.update.err)
			}
			c, w := newTestContext(http.MethodPut, "/datasources/"+testSourceID, tc.input, tc.params)

			h.Update(c)

			require.Equal(t, tc.output, w.Code)
		})
	}
}

func TestArchive(t *testing.T) {
	type mockArchive struct {
		isCalled bool
		input    string
		err      error
	}
	type mockData struct {
		archive mockArchive
	}

	tcs := map[string]struct {
		input  gin.Params
		mock   mockData
		output int
		err    error
	}{
		"success":    {input: gin.Params{{Key: "id", Value: testSourceID}}, mock: mockData{archive: mockArchive{isCalled: true, input: testSourceID}}, output: http.StatusOK},
		"wrong_path": {input: gin.Params{{Key: "id", Value: "bad"}}, output: http.StatusBadRequest},
		"uc_error":   {input: gin.Params{{Key: "id", Value: testSourceID}}, mock: mockData{archive: mockArchive{isCalled: true, input: testSourceID, err: datasource.ErrInvalidTransition}}, output: http.StatusBadRequest},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			h, uc := newTestHandler(t)
			if tc.mock.archive.isCalled {
				uc.EXPECT().Archive(context.Background(), tc.mock.archive.input).Return(tc.mock.archive.err)
			}
			c, w := newTestContext(http.MethodPost, "/datasources/"+testSourceID+"/archive", "", tc.input)

			h.Archive(c)

			require.Equal(t, tc.output, w.Code)
		})
	}
}

func TestDelete(t *testing.T) {
	type mockDelete struct {
		isCalled bool
		input    string
		err      error
	}
	type mockData struct {
		delete mockDelete
	}

	tcs := map[string]struct {
		input  gin.Params
		mock   mockData
		output int
		err    error
	}{
		"success":    {input: gin.Params{{Key: "id", Value: testSourceID}}, mock: mockData{delete: mockDelete{isCalled: true, input: testSourceID}}, output: http.StatusOK},
		"wrong_path": {input: gin.Params{{Key: "id", Value: "bad"}}, output: http.StatusBadRequest},
		"uc_error":   {input: gin.Params{{Key: "id", Value: testSourceID}}, mock: mockData{delete: mockDelete{isCalled: true, input: testSourceID, err: datasource.ErrDeleteRequiresArchived}}, output: http.StatusBadRequest},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			h, uc := newTestHandler(t)
			if tc.mock.delete.isCalled {
				uc.EXPECT().Delete(context.Background(), tc.mock.delete.input).Return(tc.mock.delete.err)
			}
			c, w := newTestContext(http.MethodDelete, "/datasources/"+testSourceID, "", tc.input)

			h.Delete(c)

			require.Equal(t, tc.output, w.Code)
		})
	}
}

func TestUpdateCrawlMode(t *testing.T) {
	type mockUpdateCrawlMode struct {
		isCalled bool
		input    datasource.UpdateCrawlModeInput
		output   datasource.UpdateCrawlModeOutput
		err      error
	}
	type mockData struct {
		update mockUpdateCrawlMode
	}

	tcs := map[string]struct {
		input  string
		params gin.Params
		mock   mockData
		output int
		err    error
	}{
		"success": {
			input:  `{"crawl_mode":"NORMAL","trigger_type":"MANUAL","reason":" reason ","event_ref":" event-1 "}`,
			params: gin.Params{{Key: "id", Value: testSourceID}},
			mock:   mockData{update: mockUpdateCrawlMode{isCalled: true, input: datasource.UpdateCrawlModeInput{ID: testSourceID, CrawlMode: string(model.CrawlModeNormal), TriggerType: string(model.TriggerTypeManual), Reason: "reason", EventRef: "event-1"}, output: datasource.UpdateCrawlModeOutput{DataSource: testDataSource()}}},
			output: http.StatusOK,
		},
		"wrong_body": {
			input:  `{`,
			params: gin.Params{{Key: "id", Value: testSourceID}},
			output: http.StatusBadRequest,
		},
		"wrong_body_validate": {
			input:  `{"crawl_mode":"BAD","trigger_type":"MANUAL"}`,
			params: gin.Params{{Key: "id", Value: testSourceID}},
			output: http.StatusBadRequest,
		},
		"uc_error": {
			input:  `{"crawl_mode":"NORMAL","trigger_type":"MANUAL"}`,
			params: gin.Params{{Key: "id", Value: testSourceID}},
			mock:   mockData{update: mockUpdateCrawlMode{isCalled: true, input: datasource.UpdateCrawlModeInput{ID: testSourceID, CrawlMode: string(model.CrawlModeNormal), TriggerType: string(model.TriggerTypeManual)}, err: datasource.ErrCrawlModeNotAllowed}},
			output: http.StatusBadRequest,
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			h, uc := newTestHandler(t)
			if tc.mock.update.isCalled {
				uc.EXPECT().UpdateCrawlMode(context.Background(), tc.mock.update.input).Return(tc.mock.update.output, tc.mock.update.err)
			}
			c, w := newTestContext(http.MethodPut, "/internal/datasources/"+testSourceID+"/crawl-mode", tc.input, tc.params)

			h.UpdateCrawlMode(c)

			require.Equal(t, tc.output, w.Code)
		})
	}
}

func TestGetActivationReadiness(t *testing.T) {
	type mockReadiness struct {
		isCalled bool
		input    datasource.ActivationReadinessInput
		output   datasource.ActivationReadinessOutput
		err      error
	}
	type mockData struct {
		readiness mockReadiness
	}

	tcs := map[string]struct {
		input  string
		params gin.Params
		mock   mockData
		output int
		err    error
	}{
		"success": {
			input:  "?command=resume",
			params: gin.Params{{Key: "project_id", Value: testProjectID}},
			mock:   mockData{readiness: mockReadiness{isCalled: true, input: datasource.ActivationReadinessInput{ProjectID: testProjectID, Command: datasource.ActivationReadinessCommandResume}, output: datasource.ActivationReadinessOutput{ProjectID: testProjectID, Command: datasource.ActivationReadinessCommandResume, DataSourceCount: 1, HasDatasource: true, CanProceed: true}}},
			output: http.StatusOK,
		},
		"wrong_path": {
			params: gin.Params{{Key: "project_id", Value: "bad"}},
			output: http.StatusBadRequest,
		},
		"wrong_query": {
			input:  "?command=bad",
			params: gin.Params{{Key: "project_id", Value: testProjectID}},
			output: http.StatusBadRequest,
		},
		"uc_error": {
			params: gin.Params{{Key: "project_id", Value: testProjectID}},
			mock:   mockData{readiness: mockReadiness{isCalled: true, input: datasource.ActivationReadinessInput{ProjectID: testProjectID}, err: datasource.ErrActivationReadinessFailed}},
			output: http.StatusBadRequest,
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			h, uc := newTestHandler(t)
			if tc.mock.readiness.isCalled {
				uc.EXPECT().GetActivationReadiness(context.Background(), tc.mock.readiness.input).Return(tc.mock.readiness.output, tc.mock.readiness.err)
			}
			c, w := newTestContext(http.MethodGet, "/internal/projects/"+testProjectID+"/activation-readiness"+tc.input, "", tc.params)

			h.GetActivationReadiness(c)

			require.Equal(t, tc.output, w.Code)
		})
	}
}

func TestActivate(t *testing.T) {
	testProjectLifecycleHandler(t, "activate", func(h *handler, c *gin.Context) { h.Activate(c) }, func(uc *datasource.MockUseCase, input string, output datasource.ProjectLifecycleOutput, err error) {
		uc.EXPECT().Activate(context.Background(), input).Return(output, err)
	})
}

func TestPause(t *testing.T) {
	testProjectLifecycleHandler(t, "pause", func(h *handler, c *gin.Context) { h.Pause(c) }, func(uc *datasource.MockUseCase, input string, output datasource.ProjectLifecycleOutput, err error) {
		uc.EXPECT().Pause(context.Background(), input).Return(output, err)
	})
}

func TestResume(t *testing.T) {
	testProjectLifecycleHandler(t, "resume", func(h *handler, c *gin.Context) { h.Resume(c) }, func(uc *datasource.MockUseCase, input string, output datasource.ProjectLifecycleOutput, err error) {
		uc.EXPECT().Resume(context.Background(), input).Return(output, err)
	})
}

func testProjectLifecycleHandler(t *testing.T, command string, call func(*handler, *gin.Context), expect func(*datasource.MockUseCase, string, datasource.ProjectLifecycleOutput, error)) {
	t.Helper()
	type mockLifecycle struct {
		isCalled bool
		input    string
		output   datasource.ProjectLifecycleOutput
		err      error
	}
	type mockData struct {
		lifecycle mockLifecycle
	}

	tcs := map[string]struct {
		input  gin.Params
		mock   mockData
		output int
		err    error
	}{
		"success":    {input: gin.Params{{Key: "project_id", Value: testProjectID}}, mock: mockData{lifecycle: mockLifecycle{isCalled: true, input: testProjectID, output: datasource.ProjectLifecycleOutput{ProjectID: testProjectID, AffectedDataSourceCount: 1}}}, output: http.StatusOK},
		"wrong_path": {input: gin.Params{{Key: "project_id", Value: "bad"}}, output: http.StatusBadRequest},
		"uc_error":   {input: gin.Params{{Key: "project_id", Value: testProjectID}}, mock: mockData{lifecycle: mockLifecycle{isCalled: true, input: testProjectID, err: datasource.ErrInvalidTransition}}, output: http.StatusBadRequest},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			h, uc := newTestHandler(t)
			if tc.mock.lifecycle.isCalled {
				expect(uc, tc.mock.lifecycle.input, tc.mock.lifecycle.output, tc.mock.lifecycle.err)
			}
			c, w := newTestContext(http.MethodPost, "/internal/projects/"+testProjectID+"/"+command, "", tc.input)

			call(h, c)

			require.Equal(t, tc.output, w.Code)
		})
	}
}

func TestUpdateProjectCrawlMode(t *testing.T) {
	type mockUpdate struct {
		isCalled bool
		input    datasource.UpdateProjectCrawlModeInput
		output   datasource.ProjectLifecycleOutput
		err      error
	}
	type mockData struct {
		update mockUpdate
	}

	tcs := map[string]struct {
		input  string
		mock   mockData
		output int
		err    error
	}{
		"success": {
			input: `{"crawl_mode":"CRISIS","trigger_type":"CRISIS_EVENT","reason":" reason ","event_ref":" event-1 "}`,
			mock: mockData{update: mockUpdate{
				isCalled: true,
				input:    datasource.UpdateProjectCrawlModeInput{ProjectID: testProjectID, CrawlMode: string(model.CrawlModeCrisis), TriggerType: string(model.TriggerTypeCrisisEvent), Reason: "reason", EventRef: "event-1"},
				output:   datasource.ProjectLifecycleOutput{ProjectID: testProjectID, AffectedDataSourceCount: 2},
			}},
			output: http.StatusOK,
		},
		"wrong_body": {
			input:  `{`,
			output: http.StatusBadRequest,
		},
		"invalid_mode": {
			input:  `{"crawl_mode":"BAD","trigger_type":"CRISIS_EVENT"}`,
			output: http.StatusBadRequest,
		},
		"uc_error": {
			input: `{"crawl_mode":"CRISIS","trigger_type":"CRISIS_EVENT"}`,
			mock: mockData{update: mockUpdate{
				isCalled: true,
				input:    datasource.UpdateProjectCrawlModeInput{ProjectID: testProjectID, CrawlMode: string(model.CrawlModeCrisis), TriggerType: string(model.TriggerTypeCrisisEvent)},
				err:      datasource.ErrUpdateFailed,
			}},
			output: http.StatusInternalServerError,
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			h, uc := newTestHandler(t)
			if tc.mock.update.isCalled {
				uc.EXPECT().UpdateProjectCrawlMode(context.Background(), tc.mock.update.input).Return(tc.mock.update.output, tc.mock.update.err)
			}
			c, w := newTestContext(http.MethodPost, "/internal/projects/"+testProjectID+"/crawl-mode", tc.input, gin.Params{{Key: "project_id", Value: testProjectID}})

			h.UpdateProjectCrawlMode(c)

			require.Equal(t, tc.output, w.Code)
		})
	}
}

func TestCreateKeywordTarget(t *testing.T) {
	testCreateTargetHandler(t, "keywords", `{"values":[" vinfast ","vinfast",""],"label":"label","platform_meta":{"platform":"tiktok"},"priority":1,"crawl_interval_minutes":10}`, datasource.CreateTargetGroupInput{DataSourceID: testSourceID, Values: []string{"vinfast"}, Label: "label", PlatformMeta: []byte(`{"platform":"tiktok"}`), Priority: 1, CrawlIntervalMinutes: 10}, func(h *handler, c *gin.Context) { h.CreateKeywordTarget(c) }, func(uc *datasource.MockUseCase, input datasource.CreateTargetGroupInput, output datasource.CreateTargetOutput, err error) {
		uc.EXPECT().CreateKeywordTarget(context.Background(), input).Return(output, err)
	})
}

func TestCreateProfileTarget(t *testing.T) {
	testCreateTargetHandler(t, "profiles", `{"values":["https://facebook.com/p"],"label":"label","crawl_interval_minutes":10}`, datasource.CreateTargetGroupInput{DataSourceID: testSourceID, Values: []string{"https://facebook.com/p"}, Label: "label", CrawlIntervalMinutes: 10}, func(h *handler, c *gin.Context) { h.CreateProfileTarget(c) }, func(uc *datasource.MockUseCase, input datasource.CreateTargetGroupInput, output datasource.CreateTargetOutput, err error) {
		uc.EXPECT().CreateProfileTarget(context.Background(), input).Return(output, err)
	})
}

func TestCreatePostTarget(t *testing.T) {
	testCreateTargetHandler(t, "posts", `{"values":["https://facebook.com/post/1"],"label":"label","crawl_interval_minutes":10}`, datasource.CreateTargetGroupInput{DataSourceID: testSourceID, Values: []string{"https://facebook.com/post/1"}, Label: "label", CrawlIntervalMinutes: 10}, func(h *handler, c *gin.Context) { h.CreatePostTarget(c) }, func(uc *datasource.MockUseCase, input datasource.CreateTargetGroupInput, output datasource.CreateTargetOutput, err error) {
		uc.EXPECT().CreatePostTarget(context.Background(), input).Return(output, err)
	})
}

func testCreateTargetHandler(t *testing.T, path string, successBody string, successInput datasource.CreateTargetGroupInput, call func(*handler, *gin.Context), expect func(*datasource.MockUseCase, datasource.CreateTargetGroupInput, datasource.CreateTargetOutput, error)) {
	t.Helper()
	type mockCreate struct {
		isCalled bool
		input    datasource.CreateTargetGroupInput
		output   datasource.CreateTargetOutput
		err      error
	}
	type mockData struct {
		create mockCreate
	}

	tcs := map[string]struct {
		input  string
		params gin.Params
		mock   mockData
		output int
		err    error
	}{
		"success": {
			input:  successBody,
			params: gin.Params{{Key: "id", Value: testSourceID}},
			mock:   mockData{create: mockCreate{isCalled: true, input: successInput, output: datasource.CreateTargetOutput{Target: testTarget()}}},
			output: http.StatusOK,
		},
		"wrong_body": {
			input:  `{`,
			params: gin.Params{{Key: "id", Value: testSourceID}},
			output: http.StatusBadRequest,
		},
		"wrong_body_validate": {
			input:  `{"values":[],"crawl_interval_minutes":10}`,
			params: gin.Params{{Key: "id", Value: testSourceID}},
			output: http.StatusBadRequest,
		},
		"uc_error": {
			input:  successBody,
			params: gin.Params{{Key: "id", Value: testSourceID}},
			mock:   mockData{create: mockCreate{isCalled: true, input: successInput, err: datasource.ErrTargetCreateFailed}},
			output: http.StatusInternalServerError,
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			h, uc := newTestHandler(t)
			if tc.mock.create.isCalled {
				expect(uc, tc.mock.create.input, tc.mock.create.output, tc.mock.create.err)
			}
			c, w := newTestContext(http.MethodPost, "/datasources/"+testSourceID+"/targets/"+path, tc.input, tc.params)

			call(h, c)

			require.Equal(t, tc.output, w.Code)
		})
	}
}

func TestListTargets(t *testing.T) {
	active := true
	type mockList struct {
		isCalled bool
		input    datasource.ListTargetsInput
		output   datasource.ListTargetsOutput
		err      error
	}
	type mockData struct {
		list mockList
	}

	tcs := map[string]struct {
		input  string
		params gin.Params
		mock   mockData
		output int
		err    error
	}{
		"success": {
			input:  "?target_type=KEYWORD&is_active=true",
			params: gin.Params{{Key: "id", Value: testSourceID}},
			mock:   mockData{list: mockList{isCalled: true, input: datasource.ListTargetsInput{DataSourceID: testSourceID, TargetType: string(model.TargetTypeKeyword), IsActive: &active}, output: datasource.ListTargetsOutput{Targets: []model.CrawlTarget{testTarget()}}}},
			output: http.StatusOK,
		},
		"wrong_query_bind": {
			input:  "?is_active=bad",
			params: gin.Params{{Key: "id", Value: testSourceID}},
			output: http.StatusBadRequest,
		},
		"wrong_query_validate": {
			input:  "?target_type=BAD",
			params: gin.Params{{Key: "id", Value: testSourceID}},
			output: http.StatusBadRequest,
		},
		"uc_error": {
			params: gin.Params{{Key: "id", Value: testSourceID}},
			mock:   mockData{list: mockList{isCalled: true, input: datasource.ListTargetsInput{DataSourceID: testSourceID}, err: datasource.ErrTargetListFailed}},
			output: http.StatusInternalServerError,
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			h, uc := newTestHandler(t)
			if tc.mock.list.isCalled {
				uc.EXPECT().ListTargets(context.Background(), tc.mock.list.input).Return(tc.mock.list.output, tc.mock.list.err)
			}
			c, w := newTestContext(http.MethodGet, "/datasources/"+testSourceID+"/targets"+tc.input, "", tc.params)

			h.ListTargets(c)

			require.Equal(t, tc.output, w.Code)
		})
	}
}

func TestDetailTarget(t *testing.T) {
	testTargetIdentityHandler(t, "detail", func(h *handler, c *gin.Context) { h.DetailTarget(c) }, func(uc *datasource.MockUseCase, input datasource.DetailTargetInput, output model.CrawlTarget, err error) {
		uc.EXPECT().DetailTarget(context.Background(), input).Return(datasource.DetailTargetOutput{Target: output}, err)
	})
}

func TestActivateTarget(t *testing.T) {
	testTargetIdentityHandler(t, "activate", func(h *handler, c *gin.Context) { h.ActivateTarget(c) }, func(uc *datasource.MockUseCase, input datasource.DetailTargetInput, output model.CrawlTarget, err error) {
		uc.EXPECT().ActivateTarget(context.Background(), datasource.ActivateTargetInput(input)).Return(datasource.ActivateTargetOutput{Target: output}, err)
	})
}

func TestDeactivateTarget(t *testing.T) {
	testTargetIdentityHandler(t, "deactivate", func(h *handler, c *gin.Context) { h.DeactivateTarget(c) }, func(uc *datasource.MockUseCase, input datasource.DetailTargetInput, output model.CrawlTarget, err error) {
		uc.EXPECT().DeactivateTarget(context.Background(), datasource.DeactivateTargetInput(input)).Return(datasource.DeactivateTargetOutput{Target: output}, err)
	})
}

func testTargetIdentityHandler(t *testing.T, path string, call func(*handler, *gin.Context), expect func(*datasource.MockUseCase, datasource.DetailTargetInput, model.CrawlTarget, error)) {
	t.Helper()
	type mockIdentity struct {
		isCalled bool
		input    datasource.DetailTargetInput
		output   model.CrawlTarget
		err      error
	}
	type mockData struct {
		identity mockIdentity
	}

	tcs := map[string]struct {
		input  gin.Params
		mock   mockData
		output int
		err    error
	}{
		"success":    {input: gin.Params{{Key: "id", Value: testSourceID}, {Key: "target_id", Value: testTargetID}}, mock: mockData{identity: mockIdentity{isCalled: true, input: datasource.DetailTargetInput{DataSourceID: testSourceID, ID: testTargetID}, output: testTarget()}}, output: http.StatusOK},
		"wrong_path": {input: gin.Params{{Key: "id", Value: "bad"}, {Key: "target_id", Value: testTargetID}}, output: http.StatusBadRequest},
		"uc_error":   {input: gin.Params{{Key: "id", Value: testSourceID}, {Key: "target_id", Value: testTargetID}}, mock: mockData{identity: mockIdentity{isCalled: true, input: datasource.DetailTargetInput{DataSourceID: testSourceID, ID: testTargetID}, err: datasource.ErrTargetNotFound}}, output: http.StatusNotFound},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			h, uc := newTestHandler(t)
			if tc.mock.identity.isCalled {
				expect(uc, tc.mock.identity.input, tc.mock.identity.output, tc.mock.identity.err)
			}
			c, w := newTestContext(http.MethodGet, "/datasources/"+testSourceID+"/targets/"+testTargetID+"/"+path, "", tc.input)

			call(h, c)

			require.Equal(t, tc.output, w.Code)
		})
	}
}

func TestUpdateTarget(t *testing.T) {
	priority := 2
	interval := 20
	type mockUpdate struct {
		isCalled bool
		input    datasource.UpdateTargetInput
		output   datasource.UpdateTargetOutput
		err      error
	}
	type mockData struct {
		update mockUpdate
	}

	tcs := map[string]struct {
		input  string
		params gin.Params
		mock   mockData
		output int
		err    error
	}{
		"success": {
			input:  `{"values":[" a ",""],"label":"label","platform_meta":{"k":"v"},"priority":2,"crawl_interval_minutes":20}`,
			params: gin.Params{{Key: "id", Value: testSourceID}, {Key: "target_id", Value: testTargetID}},
			mock:   mockData{update: mockUpdate{isCalled: true, input: datasource.UpdateTargetInput{DataSourceID: testSourceID, ID: testTargetID, Values: []string{"a"}, Label: "label", PlatformMeta: []byte(`{"k":"v"}`), Priority: &priority, CrawlIntervalMinutes: &interval}, output: datasource.UpdateTargetOutput{Target: testTarget()}}},
			output: http.StatusOK,
		},
		"wrong_body": {
			input:  `{`,
			params: gin.Params{{Key: "id", Value: testSourceID}, {Key: "target_id", Value: testTargetID}},
			output: http.StatusBadRequest,
		},
		"wrong_body_validate": {
			input:  `{"crawl_interval_minutes":0}`,
			params: gin.Params{{Key: "id", Value: testSourceID}, {Key: "target_id", Value: testTargetID}},
			output: http.StatusBadRequest,
		},
		"uc_error": {
			input:  `{}`,
			params: gin.Params{{Key: "id", Value: testSourceID}, {Key: "target_id", Value: testTargetID}},
			mock:   mockData{update: mockUpdate{isCalled: true, input: datasource.UpdateTargetInput{DataSourceID: testSourceID, ID: testTargetID}, err: datasource.ErrTargetUpdateFailed}},
			output: http.StatusInternalServerError,
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			h, uc := newTestHandler(t)
			if tc.mock.update.isCalled {
				uc.EXPECT().UpdateTarget(context.Background(), tc.mock.update.input).Return(tc.mock.update.output, tc.mock.update.err)
			}
			c, w := newTestContext(http.MethodPut, "/datasources/"+testSourceID+"/targets/"+testTargetID, tc.input, tc.params)

			h.UpdateTarget(c)

			require.Equal(t, tc.output, w.Code)
		})
	}
}

func TestDeleteTarget(t *testing.T) {
	type mockDelete struct {
		isCalled bool
		input    datasource.DeleteTargetInput
		err      error
	}
	type mockData struct {
		delete mockDelete
	}

	tcs := map[string]struct {
		input  gin.Params
		mock   mockData
		output int
		err    error
	}{
		"success":    {input: gin.Params{{Key: "id", Value: testSourceID}, {Key: "target_id", Value: testTargetID}}, mock: mockData{delete: mockDelete{isCalled: true, input: datasource.DeleteTargetInput{DataSourceID: testSourceID, ID: testTargetID}}}, output: http.StatusOK},
		"wrong_path": {input: gin.Params{{Key: "id", Value: "bad"}, {Key: "target_id", Value: testTargetID}}, output: http.StatusBadRequest},
		"uc_error":   {input: gin.Params{{Key: "id", Value: testSourceID}, {Key: "target_id", Value: testTargetID}}, mock: mockData{delete: mockDelete{isCalled: true, input: datasource.DeleteTargetInput{DataSourceID: testSourceID, ID: testTargetID}, err: datasource.ErrTargetDeleteFailed}}, output: http.StatusInternalServerError},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			h, uc := newTestHandler(t)
			if tc.mock.delete.isCalled {
				uc.EXPECT().DeleteTarget(context.Background(), tc.mock.delete.input).Return(tc.mock.delete.err)
			}
			c, w := newTestContext(http.MethodDelete, "/datasources/"+testSourceID+"/targets/"+testTargetID, "", tc.input)

			h.DeleteTarget(c)

			require.Equal(t, tc.output, w.Code)
		})
	}
}
