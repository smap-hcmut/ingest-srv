package usecase

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"ingest-srv/internal/datasource"
	"ingest-srv/internal/dryrun"
	producer "ingest-srv/internal/dryrun/delivery/rabbitmq/producer"
	dryrunRepo "ingest-srv/internal/dryrun/repository"
	"ingest-srv/internal/model"

	"github.com/smap-hcmut/shared-libs/go/log"
	"github.com/smap-hcmut/shared-libs/go/paginator"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const (
	testSourceID  = "550e8400-e29b-41d4-a716-446655440000"
	testProjectID = "550e8400-e29b-41d4-a716-446655440001"
	testTargetID  = "550e8400-e29b-41d4-a716-446655440002"
)

func newDryrunUC(t *testing.T) (*implUseCase, *dryrunRepo.MockRepository, *datasource.MockUseCase, *producer.MockProducer) {
	t.Helper()
	l := log.NewZapLogger(log.ZapConfig{Level: log.LevelFatal, Mode: log.ModeProduction, Encoding: log.EncodingJSON})
	r := dryrunRepo.NewMockRepository(t)
	dsUC := datasource.NewMockUseCase(t)
	pub := producer.NewMockProducer(t)
	return &implUseCase{
		l:         l,
		repo:      r,
		dsUC:      dsUC,
		publisher: pub,
		now:       func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) },
	}, r, dsUC, pub
}

func TestNew(t *testing.T) {
	tcs := map[string]struct {
		input  struct{}
		mock   struct{}
		output dryrun.UseCase
		err    error
	}{
		"success": {},
	}

	for name := range tcs {
		t.Run(name, func(t *testing.T) {
			uc, r, dsUC, pub := newDryrunUC(t)

			got := New(uc.l, r, dsUC, nil, pub)

			require.NotNil(t, got)
		})
	}
}

func dryrunSource(status model.SourceStatus) model.DataSource {
	return model.DataSource{
		ID:             testSourceID,
		ProjectID:      testProjectID,
		SourceType:     model.SourceTypeTikTok,
		SourceCategory: model.SourceCategoryCrawl,
		Status:         status,
	}
}

func dryrunTarget() model.CrawlTarget {
	return model.CrawlTarget{ID: testTargetID, DataSourceID: testSourceID, TargetType: model.TargetTypeKeyword, Values: []string{"vinfast"}}
}

func dryrunResult(status model.DryrunStatus) model.DryrunResult {
	return model.DryrunResult{ID: "result-1", SourceID: testSourceID, ProjectID: testProjectID, TargetID: testTargetID, JobID: "task-1", Status: status}
}

func TestTrigger(t *testing.T) {
	limit := 5
	type mockTrigger struct {
		detailCalled       bool
		source             model.DataSource
		targetCalled       bool
		target             model.CrawlTarget
		latestCalled       bool
		latest             model.DryrunResult
		latestErr          error
		createCalled       bool
		createResult       model.DryrunResult
		createErr          error
		markCalled         bool
		markResult         datasource.MarkDryrunRunningOutput
		publishCalled      bool
		publishErr         error
		completeFailCalled bool
	}

	tcs := map[string]struct {
		input  dryrun.TriggerInput
		mock   mockTrigger
		output dryrun.TriggerOutput
		err    error
	}{
		"success": {
			input:  dryrun.TriggerInput{SourceID: testSourceID, TargetID: testTargetID, SampleLimit: &limit},
			mock:   mockTrigger{detailCalled: true, source: dryrunSource(model.SourceStatusReady), targetCalled: true, target: dryrunTarget(), latestCalled: true, latestErr: dryrunRepo.ErrNotFound, createCalled: true, createResult: dryrunResult(model.DryrunStatusRunning), markCalled: true, markResult: datasource.MarkDryrunRunningOutput{DataSource: dryrunSource(model.SourceStatusPending)}, publishCalled: true},
			output: dryrun.TriggerOutput{Result: dryrunResult(model.DryrunStatusRunning), DataSource: dryrunSource(model.SourceStatusPending)},
		},
		"invalid_input": {
			err: dryrun.ErrSourceNotFound,
		},
		"invalid_source_status": {
			input: dryrun.TriggerInput{SourceID: testSourceID, TargetID: testTargetID},
			mock:  mockTrigger{detailCalled: true, source: dryrunSource(model.SourceStatusActive)},
			err:   dryrun.ErrDryrunNotAllowed,
		},
		"target_required": {
			input: dryrun.TriggerInput{SourceID: testSourceID},
			mock:  mockTrigger{detailCalled: true, source: dryrunSource(model.SourceStatusReady)},
			err:   dryrun.ErrTargetRequired,
		},
		"already_running": {
			input: dryrun.TriggerInput{SourceID: testSourceID, TargetID: testTargetID},
			mock:  mockTrigger{detailCalled: true, source: dryrunSource(model.SourceStatusReady), targetCalled: true, target: dryrunTarget(), latestCalled: true, latest: dryrunResult(model.DryrunStatusRunning)},
			err:   dryrun.ErrDryrunAlreadyRunning,
		},
		"dispatch_failed": {
			input: dryrun.TriggerInput{SourceID: testSourceID, TargetID: testTargetID},
			mock:  mockTrigger{detailCalled: true, source: dryrunSource(model.SourceStatusReady), targetCalled: true, target: dryrunTarget(), latestCalled: true, latestErr: dryrunRepo.ErrNotFound, createCalled: true, createResult: dryrunResult(model.DryrunStatusRunning), markCalled: true, markResult: datasource.MarkDryrunRunningOutput{DataSource: dryrunSource(model.SourceStatusPending)}, publishCalled: true, publishErr: dryrun.ErrDispatchFailed, completeFailCalled: true},
			err:   dryrun.ErrDispatchFailed,
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			uc, r, dsUC, pub := newDryrunUC(t)
			if tc.mock.detailCalled {
				dsUC.EXPECT().Detail(context.Background(), testSourceID).Return(datasource.DetailOutput{DataSource: tc.mock.source}, nil)
			}
			if tc.mock.targetCalled {
				dsUC.EXPECT().DetailTarget(context.Background(), datasource.DetailTargetInput{DataSourceID: testSourceID, ID: testTargetID}).Return(datasource.DetailTargetOutput{Target: tc.mock.target}, nil)
			}
			if tc.mock.latestCalled {
				r.EXPECT().GetLatest(context.Background(), dryrunRepo.GetLatestOptions{SourceID: testSourceID, TargetID: testTargetID}).Return(tc.mock.latest, tc.mock.latestErr)
			}
			if tc.mock.createCalled {
				r.EXPECT().CreateResult(context.Background(), mock.MatchedBy(func(opt dryrunRepo.CreateResultOptions) bool {
					return opt.SourceID == testSourceID && opt.ProjectID == testProjectID && opt.TargetID == testTargetID && opt.Status == string(model.DryrunStatusRunning)
				})).Return(tc.mock.createResult, tc.mock.createErr)
			}
			if tc.mock.markCalled {
				dsUC.EXPECT().MarkDryrunRunning(context.Background(), mock.MatchedBy(func(input datasource.MarkDryrunRunningInput) bool {
					return input.ID == testSourceID && input.DryrunLastResultID == "result-1"
				})).Return(tc.mock.markResult, nil)
			}
			if tc.mock.publishCalled {
				pub.EXPECT().PublishDispatch(context.Background(), mock.MatchedBy(func(input dryrun.PublishDispatchInput) bool {
					return input.Queue == dryrun.QueueNameTikTokTasks && input.Action == dryrun.ActionNameFullFlow
				})).Return(tc.mock.publishErr)
			}
			if tc.mock.completeFailCalled {
				r.EXPECT().CompleteResult(context.Background(), mock.MatchedBy(func(opt dryrunRepo.CompleteResultOptions) bool {
					return opt.ID == "result-1" && opt.Status == string(model.DryrunStatusFailed)
				})).Return(dryrunResult(model.DryrunStatusFailed), dryrunSource(model.SourceStatusPending), nil)
			}

			output, err := uc.Trigger(context.Background(), tc.input)

			require.ErrorIs(t, err, tc.err)
			if tc.err == nil {
				require.Equal(t, tc.output, output)
			}
		})
	}
}

func TestGetLatest(t *testing.T) {
	type mockLatest struct {
		isCalled bool
		input    dryrunRepo.GetLatestOptions
		output   model.DryrunResult
		err      error
	}

	tcs := map[string]struct {
		input  dryrun.GetLatestInput
		mock   mockLatest
		output dryrun.GetLatestOutput
		err    error
	}{
		"success":    {input: dryrun.GetLatestInput{SourceID: " " + testSourceID + " ", TargetID: " " + testTargetID + " "}, mock: mockLatest{isCalled: true, input: dryrunRepo.GetLatestOptions{SourceID: testSourceID, TargetID: testTargetID}, output: dryrunResult(model.DryrunStatusSuccess)}, output: dryrun.GetLatestOutput{Result: dryrunResult(model.DryrunStatusSuccess)}},
		"invalid":    {err: dryrun.ErrSourceNotFound},
		"not_found":  {input: dryrun.GetLatestInput{SourceID: testSourceID}, mock: mockLatest{isCalled: true, input: dryrunRepo.GetLatestOptions{SourceID: testSourceID}, err: dryrunRepo.ErrNotFound}, err: dryrun.ErrResultNotFound},
		"repo_error": {input: dryrun.GetLatestInput{SourceID: testSourceID}, mock: mockLatest{isCalled: true, input: dryrunRepo.GetLatestOptions{SourceID: testSourceID}, err: dryrunRepo.ErrFailedToGet}, err: dryrun.ErrGetFailed},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			uc, r, _, _ := newDryrunUC(t)
			if tc.mock.isCalled {
				r.EXPECT().GetLatest(context.Background(), tc.mock.input).Return(tc.mock.output, tc.mock.err)
			}

			output, err := uc.GetLatest(context.Background(), tc.input)

			require.ErrorIs(t, err, tc.err)
			require.Equal(t, tc.output, output)
		})
	}
}

func TestListHistory(t *testing.T) {
	type mockHistory struct {
		isCalled bool
		input    dryrunRepo.ListHistoryOptions
		output   []model.DryrunResult
		pag      paginator.Paginator
		err      error
	}

	tcs := map[string]struct {
		input  dryrun.ListHistoryInput
		mock   mockHistory
		output dryrun.ListHistoryOutput
		err    error
	}{
		"success":    {input: dryrun.ListHistoryInput{SourceID: " " + testSourceID + " ", TargetID: testTargetID, Paginator: paginator.PaginateQuery{Page: 1, Limit: 10}}, mock: mockHistory{isCalled: true, input: dryrunRepo.ListHistoryOptions{SourceID: testSourceID, TargetID: testTargetID, Paginator: paginator.PaginateQuery{Page: 1, Limit: 10}}, output: []model.DryrunResult{dryrunResult(model.DryrunStatusSuccess)}, pag: paginator.Paginator{Total: 1}}, output: dryrun.ListHistoryOutput{Results: []model.DryrunResult{dryrunResult(model.DryrunStatusSuccess)}, Paginator: paginator.Paginator{Total: 1}}},
		"invalid":    {err: dryrun.ErrSourceNotFound},
		"repo_error": {input: dryrun.ListHistoryInput{SourceID: testSourceID}, mock: mockHistory{isCalled: true, input: dryrunRepo.ListHistoryOptions{SourceID: testSourceID, Paginator: paginator.PaginateQuery{Page: 1, Limit: 15}}, err: dryrunRepo.ErrFailedToGet}, err: dryrun.ErrListFailed},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			uc, r, _, _ := newDryrunUC(t)
			if tc.mock.isCalled {
				r.EXPECT().ListHistory(context.Background(), tc.mock.input).Return(tc.mock.output, tc.mock.pag, tc.mock.err)
			}

			output, err := uc.ListHistory(context.Background(), tc.input)

			require.ErrorIs(t, err, tc.err)
			require.Equal(t, tc.output, output)
		})
	}
}

func TestHandleCompletion(t *testing.T) {
	completedAt := "2026-01-01T00:00:00Z"
	tmp, err := os.CreateTemp(t.TempDir(), "dryrun-*.json")
	require.NoError(t, err)
	_, err = tmp.WriteString(`{"params":{"limit":2},"result":[{"id":1},{"id":2},{"id":3}],"item_count":3}`)
	require.NoError(t, err)
	require.NoError(t, tmp.Close())

	type mockCompletion struct {
		getCalled      bool
		getOutput      model.DryrunResult
		getErr         error
		completeCalled bool
		completeInput  dryrunRepo.CompleteResultOptions
		completeErr    error
	}

	tcs := map[string]struct {
		input  dryrun.HandleCompletionInput
		mock   mockCompletion
		output struct{}
		err    error
	}{
		"success": {
			input: dryrun.HandleCompletionInput{TaskID: "task-1", Status: "success", StorageBucket: "local", StoragePath: tmp.Name(), CompletedAt: completedAt},
			mock:  mockCompletion{getCalled: true, getOutput: dryrunResult(model.DryrunStatusRunning), completeCalled: true, completeInput: dryrunRepo.CompleteResultOptions{ID: "result-1", Status: string(model.DryrunStatusSuccess), SampleCount: 2, CompletedAt: mustParseTimePtr(completedAt), TotalFound: intPtr(3), SampleData: []byte(`[{"id":1},{"id":2}]`), ActivateTarget: true}},
		},
		"invalid": {
			input: dryrun.HandleCompletionInput{},
			err:   dryrun.ErrInvalidCompletionInput,
		},
		"not_found": {
			input: dryrun.HandleCompletionInput{TaskID: "task-1", Status: "error"},
			mock:  mockCompletion{getCalled: true, getErr: dryrunRepo.ErrNotFound},
			err:   dryrun.ErrCompletionTaskNotFound,
		},
		"error_status": {
			input: dryrun.HandleCompletionInput{TaskID: "task-1", Status: "error", Error: "boom", CompletedAt: completedAt},
			mock:  mockCompletion{getCalled: true, getOutput: dryrunResult(model.DryrunStatusRunning), completeCalled: true, completeInput: dryrunRepo.CompleteResultOptions{ID: "result-1", Status: string(model.DryrunStatusFailed), SampleCount: 0, CompletedAt: mustParseTimePtr(completedAt), ErrorMessage: "boom"}},
		},
		"duplicate_terminal": {
			input: dryrun.HandleCompletionInput{TaskID: "task-1", Status: "error"},
			mock:  mockCompletion{getCalled: true, getOutput: model.DryrunResult{ID: "result-1", SourceID: testSourceID, ProjectID: testProjectID, JobID: "task-1", Status: model.DryrunStatusSuccess}},
		},
		"download_failed": {
			input: dryrun.HandleCompletionInput{TaskID: "task-1", Status: "success", StorageBucket: "local", StoragePath: "/missing", CompletedAt: completedAt},
			mock:  mockCompletion{getCalled: true, getOutput: dryrunResult(model.DryrunStatusRunning), completeCalled: true, completeInput: dryrunRepo.CompleteResultOptions{ID: "result-1", Status: string(model.DryrunStatusFailed), SampleCount: 0, CompletedAt: mustParseTimePtr(completedAt), ErrorMessage: "download dryrun artifact: open /missing: no such file or directory"}},
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			uc, r, _, _ := newDryrunUC(t)
			if tc.mock.getCalled {
				r.EXPECT().GetByJobID(context.Background(), "task-1").Return(tc.mock.getOutput, tc.mock.getErr)
			}
			if tc.mock.completeCalled {
				r.EXPECT().CompleteResult(context.Background(), mock.MatchedBy(func(opt dryrunRepo.CompleteResultOptions) bool {
					if opt.ID != tc.mock.completeInput.ID || opt.Status != tc.mock.completeInput.Status || opt.SampleCount != tc.mock.completeInput.SampleCount || opt.ErrorMessage != tc.mock.completeInput.ErrorMessage || opt.ActivateTarget != tc.mock.completeInput.ActivateTarget {
						return false
					}
					if tc.mock.completeInput.TotalFound != nil && (opt.TotalFound == nil || *opt.TotalFound != *tc.mock.completeInput.TotalFound) {
						return false
					}
					return true
				})).Return(model.DryrunResult{}, model.DataSource{}, tc.mock.completeErr)
			}

			err := uc.HandleCompletion(context.Background(), tc.input)

			require.ErrorIs(t, err, tc.err)
		})
	}
}

func TestDryrunFlowHelpers(t *testing.T) {
	tcs := map[string]struct {
		input  string
		mock   struct{}
		output struct{}
		err    error
	}{
		"success": {},
	}

	for name := range tcs {
		t.Run(name, func(t *testing.T) {
			uc, _, _, _ := newDryrunUC(t)

			require.Equal(t, map[string]interface{}{"parse_ids": []string{"1", "2"}, "limit": 5, dryrun.ParamKeyRuntimeKind: string(dryrun.RuntimeKindDryrun)}, uc.buildPostDetailParams([]string{"1", "2"}, 5))
			require.Equal(t, intPtr(10), uc.normalizeTotalFoundFromObject(map[string]interface{}{"total_posts": float64(10)}, nil))
			require.Equal(t, intPtr(7), uc.normalizeTotalFoundFromObject(map[string]interface{}{"item_count": int64(7)}, nil))
			require.Equal(t, 2, uc.parseInt(json.Number("2")))
			require.Equal(t, []string{"1", "2"}, mustParseFacebookIDs(t, uc, []byte(`{"parse_ids":["1"," 2 ",""]}`)))
			uc.publisher = nil
			require.ErrorIs(t, uc.publishDispatch(context.Background(), dryrun.PublishDispatchInput{}), dryrun.ErrDispatchFailed)
			got, err := uc.readAllAndClose(io.NopCloser(strings.NewReader("abc")))
			require.NoError(t, err)
			require.Equal(t, []byte("abc"), got)
			_, err = uc.readAllAndClose(nil)
			require.Error(t, err)
		})
	}
}

func TestBuildDispatchSpec(t *testing.T) {
	tcs := map[string]struct {
		input struct {
			source model.DataSource
			target *model.CrawlTarget
		}
		mock   struct{}
		output dryrun.QueueName
		err    error
	}{
		"tiktok_multi_keyword": {
			input: struct {
				source model.DataSource
				target *model.CrawlTarget
			}{source: dryrunSource(model.SourceStatusReady), target: &model.CrawlTarget{ID: testTargetID, TargetType: model.TargetTypeKeyword, Values: []string{"a", "b"}}},
			output: dryrun.QueueNameTikTokTasks,
		},
		"facebook_post": {
			input: struct {
				source model.DataSource
				target *model.CrawlTarget
			}{source: func() model.DataSource {
				s := dryrunSource(model.SourceStatusReady)
				s.SourceType = model.SourceTypeFacebook
				return s
			}(), target: &model.CrawlTarget{ID: testTargetID, TargetType: model.TargetTypePostURL, PlatformMeta: []byte(`{"parse_ids":["1"]}`)}},
			output: dryrun.QueueNameFacebookTasks,
		},
		"unsupported": {
			input: struct {
				source model.DataSource
				target *model.CrawlTarget
			}{source: func() model.DataSource {
				s := dryrunSource(model.SourceStatusReady)
				s.SourceType = model.SourceTypeYouTube
				return s
			}(), target: &model.CrawlTarget{ID: testTargetID, TargetType: model.TargetTypeKeyword, Values: []string{"a"}}},
			err: dryrun.ErrUnsupportedMapping,
		},
		"facebook_bad_meta": {
			input: struct {
				source model.DataSource
				target *model.CrawlTarget
			}{source: func() model.DataSource {
				s := dryrunSource(model.SourceStatusReady)
				s.SourceType = model.SourceTypeFacebook
				return s
			}(), target: &model.CrawlTarget{ID: testTargetID, TargetType: model.TargetTypePostURL, PlatformMeta: []byte(`{}`)}},
			err: dryrun.ErrUnsupportedMapping,
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			uc, _, _, _ := newDryrunUC(t)

			spec, warnings, err := uc.buildDispatchSpec(tc.input.source, tc.input.target, 5)

			require.ErrorIs(t, err, tc.err)
			if tc.err == nil {
				require.Equal(t, tc.output, spec.Queue)
			}
			if name == "tiktok_multi_keyword" {
				require.NotEmpty(t, warnings)
				require.Equal(t, string(dryrun.WarningCodeMultiValueKeyword), spec.Params[dryrun.ParamKeyDryrunWarningCode])
			}
		})
	}
}

func TestBuildSuccessUpdate(t *testing.T) {
	tcs := map[string]struct {
		input  []byte
		mock   struct{}
		output model.DryrunStatus
		err    error
	}{
		"invalid_json":                       {input: []byte(`{`), output: model.DryrunStatusWarning},
		"empty_result":                       {input: []byte(`{"result":[]}`), output: model.DryrunStatusSuccess},
		"object_fallback_with_warning_param": {input: []byte(`{"params":{"dryrun_warning_code":"x","dryrun_warning_message":"y"},"result":{"id":1}}`), output: model.DryrunStatusWarning},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			uc, _, _, _ := newDryrunUC(t)

			opt, status, err := uc.buildSuccessUpdate(tc.input, nil)

			require.ErrorIs(t, err, tc.err)
			require.Equal(t, tc.output, status)
			if tc.output == model.DryrunStatusWarning {
				require.NotEmpty(t, opt.Warnings)
			}
		})
	}
}

func TestApplyDatasourceResult(t *testing.T) {
	tcs := map[string]struct {
		input  model.DryrunStatus
		mock   datasource.ApplyDryrunResultOutput
		output model.DataSource
		err    error
	}{
		"success": {input: model.DryrunStatusSuccess, mock: datasource.ApplyDryrunResultOutput{DataSource: dryrunSource(model.SourceStatusReady)}, output: dryrunSource(model.SourceStatusReady)},
		"error":   {input: model.DryrunStatusFailed, err: dryrun.ErrUpdateFailed},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			uc, _, dsUC, _ := newDryrunUC(t)
			dsUC.EXPECT().ApplyDryrunResult(context.Background(), datasource.ApplyDryrunResultInput{ID: testSourceID, DryrunLastResultID: "result-1", DryrunStatus: string(tc.input)}).Return(tc.mock, tc.err)

			output, err := uc.applyDatasourceResult(context.Background(), testSourceID, "result-1", tc.input)

			require.ErrorIs(t, err, tc.err)
			require.Equal(t, tc.output, output)
		})
	}
}

func TestEnsureActivatedTargetAfterUsableDryrun(t *testing.T) {
	tcs := map[string]struct {
		input  model.DryrunResult
		mock   datasource.ActivateTargetOutput
		output struct{}
		err    error
	}{
		"skip_non_usable": {input: dryrunResult(model.DryrunStatusFailed)},
		"skip_no_target":  {input: model.DryrunResult{ID: "result-1", SourceID: testSourceID, Status: model.DryrunStatusSuccess}},
		"success":         {input: dryrunResult(model.DryrunStatusSuccess), mock: datasource.ActivateTargetOutput{Target: dryrunTarget()}},
		"allowed_skip":    {input: dryrunResult(model.DryrunStatusSuccess)},
		"hard_error":      {input: dryrunResult(model.DryrunStatusSuccess), err: dryrun.ErrUpdateFailed},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			uc, _, dsUC, _ := newDryrunUC(t)
			if name == "success" || name == "allowed_skip" || name == "hard_error" {
				callErr := error(nil)
				if name == "allowed_skip" {
					callErr = datasource.ErrTargetActivateNotAllowed
				}
				if name == "hard_error" {
					callErr = datasource.ErrUpdateFailed
				}
				dsUC.EXPECT().ActivateTarget(context.Background(), datasource.ActivateTargetInput{DataSourceID: testSourceID, ID: testTargetID}).Return(tc.mock, callErr)
			}

			err := uc.ensureActivatedTargetAfterUsableDryrun(context.Background(), tc.input)

			require.ErrorIs(t, err, tc.err)
		})
	}
}

func mustParseFacebookIDs(t *testing.T, uc *implUseCase, raw []byte) []string {
	t.Helper()
	ids, err := uc.parseFacebookParseIDs(raw)
	require.NoError(t, err)
	return ids
}

func mustParseTimePtr(raw string) *time.Time {
	parsed, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		panic(err)
	}
	parsed = parsed.UTC()
	return &parsed
}
