package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"ingest-srv/internal/execution"
	producer "ingest-srv/internal/execution/delivery/rabbitmq/producer"
	repo "ingest-srv/internal/execution/repository"
	"ingest-srv/internal/model"
	"ingest-srv/internal/uap"
	"ingest-srv/pkg/microservice"

	"github.com/smap-hcmut/shared-libs/go/log"
	sharedMinio "github.com/smap-hcmut/shared-libs/go/minio"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const (
	testSourceID  = "550e8400-e29b-41d4-a716-446655440000"
	testProjectID = "550e8400-e29b-41d4-a716-446655440001"
	testTargetID  = "550e8400-e29b-41d4-a716-446655440002"
)

func newExecutionUC(t *testing.T) (*implUseCase, *repo.MockRepository, *producer.MockProducer, *microservice.MockProjectUseCase) {
	t.Helper()
	l := log.NewZapLogger(log.ZapConfig{Level: log.LevelFatal, Mode: log.ModeProduction, Encoding: log.EncodingJSON})
	r := repo.NewMockRepository(t)
	pub := producer.NewMockProducer(t)
	project := microservice.NewMockProjectUseCase(t)
	return &implUseCase{
		l:         l,
		repo:      r,
		publisher: pub,
		project:   project,
		now:       func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) },
		sleep:     func(time.Duration) {},
	}, r, pub, project
}

func executionSource() model.DataSource {
	mode := model.CrawlModeNormal
	return model.DataSource{ID: testSourceID, ProjectID: testProjectID, SourceType: model.SourceTypeTikTok, SourceCategory: model.SourceCategoryCrawl, Status: model.SourceStatusActive, CrawlMode: &mode}
}

func executionTarget() model.CrawlTarget {
	return model.CrawlTarget{ID: testTargetID, DataSourceID: testSourceID, TargetType: model.TargetTypeKeyword, Values: []string{"vinfast"}, IsActive: true, CrawlIntervalMinutes: 10}
}

func TestNew(t *testing.T) {
	tcs := map[string]struct {
		input  struct{}
		mock   struct{}
		output execution.UseCase
		err    error
	}{
		"success": {},
	}

	for name := range tcs {
		t.Run(name, func(t *testing.T) {
			uc, r, pub, project := newExecutionUC(t)

			got := New(uc.l, r, nil, pub, nil, project)

			require.NotNil(t, got)
		})
	}
}

func TestDispatchTarget(t *testing.T) {
	ctx := repo.DispatchContext{Source: executionSource(), Target: executionTarget()}
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	type mockDispatch struct {
		getCalled      bool
		getOutput      repo.DispatchContext
		getErr         error
		createJob      bool
		createTask     bool
		publish        bool
		publishErr     error
		markPublished  bool
		finalizeCalled bool
	}

	tcs := map[string]struct {
		input  execution.DispatchTargetInput
		mock   mockDispatch
		output string
		err    error
	}{
		"success": {
			input:  execution.DispatchTargetInput{DataSourceID: testSourceID, TargetID: testTargetID, RequestedAt: now, ScheduledFor: now},
			mock:   mockDispatch{getCalled: true, getOutput: ctx, createJob: true, createTask: true, publish: true, markPublished: true},
			output: string(model.JobStatusRunning),
		},
		"repo_not_found": {
			input: execution.DispatchTargetInput{DataSourceID: testSourceID, TargetID: testTargetID},
			mock:  mockDispatch{getCalled: true, getErr: repo.ErrTargetNotFound},
			err:   execution.ErrTargetNotFound,
		},
		"source_not_found": {
			input: execution.DispatchTargetInput{DataSourceID: testSourceID, TargetID: testTargetID},
			mock:  mockDispatch{getCalled: true, getErr: repo.ErrDataSourceNotFound},
			err:   execution.ErrDataSourceNotFound,
		},
		"not_allowed_context": {
			input: execution.DispatchTargetInput{DataSourceID: testSourceID, TargetID: testTargetID},
			mock: mockDispatch{getCalled: true, getOutput: func() repo.DispatchContext {
				c := ctx
				c.Target.IsActive = false
				return c
			}()},
			err: execution.ErrDispatchNotAllowed,
		},
		"unsupported_mapping": {
			input: execution.DispatchTargetInput{DataSourceID: testSourceID, TargetID: testTargetID},
			mock: mockDispatch{getCalled: true, getOutput: func() repo.DispatchContext {
				c := ctx
				c.Source.SourceType = model.SourceTypeWebhook
				return c
			}()},
			err: execution.ErrUnsupportedDispatchMapping,
		},
		"publish_failed": {
			input:  execution.DispatchTargetInput{DataSourceID: testSourceID, TargetID: testTargetID, RequestedAt: now, ScheduledFor: now},
			mock:   mockDispatch{getCalled: true, getOutput: ctx, createJob: true, createTask: true, publish: true, publishErr: execution.ErrDispatchFailed, finalizeCalled: true},
			output: string(model.JobStatusFailed),
			err:    execution.ErrDispatchFailed,
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			uc, r, pub, project := newExecutionUC(t)
			if tc.mock.getCalled {
				r.EXPECT().GetDispatchContext(context.Background(), testSourceID, testTargetID).Return(tc.mock.getOutput, tc.mock.getErr)
			}
			if tc.mock.createJob {
				r.EXPECT().CreateScheduledJob(context.Background(), mock.AnythingOfType("repository.CreateScheduledJobOptions")).Return(model.ScheduledJob{ID: "job-1"}, nil)
			}
			if tc.mock.createTask {
				project.EXPECT().Detail(context.Background(), testProjectID).Return(microservice.ProjectDetail{ID: testProjectID, Status: microservice.ProjectStatusActive, DomainTypeCode: "domain"}, nil)
				r.EXPECT().CreateExternalTask(context.Background(), mock.MatchedBy(func(opt repo.CreateExternalTaskOptions) bool {
					return opt.Source.ID == testSourceID && opt.Target.ID == testTargetID && opt.ScheduledJobID == "job-1" && opt.DomainTypeCode == "domain"
				})).Return(model.ExternalTask{ID: "external-1"}, nil)
			}
			if tc.mock.publish {
				pub.EXPECT().PublishDispatch(context.Background(), mock.MatchedBy(func(input execution.PublishDispatchInput) bool {
					return input.Action == execution.ActionNameFullFlow
				})).Return(tc.mock.publishErr)
			}
			if tc.mock.markPublished {
				r.EXPECT().MarkExternalTaskPublished(context.Background(), mock.MatchedBy(func(opt repo.MarkExternalTaskPublishedOptions) bool {
					return opt.ExternalTaskID == "external-1"
				})).Return(nil)
			}
			if tc.mock.publishErr != nil {
				r.EXPECT().MarkExternalTaskFailed(context.Background(), mock.AnythingOfType("repository.MarkExternalTaskFailedOptions")).Return(nil)
			}
			if tc.mock.finalizeCalled {
				r.EXPECT().FinalizeScheduledJob(context.Background(), mock.MatchedBy(func(opt repo.FinalizeScheduledJobOptions) bool {
					return opt.ScheduledJobID == "job-1" && opt.Status == model.JobStatusFailed
				})).Return(nil)
			}

			output, err := uc.DispatchTarget(context.Background(), tc.input)

			require.ErrorIs(t, err, tc.err)
			if tc.output != "" {
				require.Equal(t, tc.output, output.Status)
			}
		})
	}
}

func TestDispatchTargetManually(t *testing.T) {
	tcs := map[string]struct {
		input  execution.DispatchTargetManuallyInput
		mock   struct{}
		output execution.DispatchTargetOutput
		err    error
	}{
		"repo_error": {input: execution.DispatchTargetManuallyInput{DataSourceID: testSourceID, TargetID: testTargetID}, err: execution.ErrTargetNotFound},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			uc, r, _, _ := newExecutionUC(t)
			r.EXPECT().GetDispatchContext(context.Background(), testSourceID, testTargetID).Return(repo.DispatchContext{}, repo.ErrTargetNotFound)

			output, err := uc.DispatchTargetManually(context.Background(), tc.input)

			require.ErrorIs(t, err, tc.err)
			require.Equal(t, tc.output, output)
		})
	}
}

func TestDispatchPrepared(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	ctx := repo.DispatchContext{Source: executionSource(), Target: executionTarget()}
	goodSpec := execution.DispatchSpec{Queue: execution.QueueName("q"), Action: execution.ActionNameFullFlow, Keyword: "vinfast", Params: map[string]interface{}{"keyword": "vinfast"}}
	badSpec := execution.DispatchSpec{Queue: execution.QueueName("q"), Action: execution.ActionNameFullFlow, Keyword: "bad", Params: map[string]interface{}{"bad": func() {}}}

	tcs := map[string]struct {
		input struct {
			specs []execution.DispatchSpec
		}
		mock struct {
			createJobErr error
			publishErr   error
			finalizeErr  error
		}
		output string
		err    error
	}{
		"empty_specs": {
			err: execution.ErrDispatchNotAllowed,
		},
		"create_job_conflict": {
			input: struct{ specs []execution.DispatchSpec }{specs: []execution.DispatchSpec{goodSpec}},
			mock:  struct{ createJobErr, publishErr, finalizeErr error }{createJobErr: repo.ErrDispatchConflict},
			err:   execution.ErrDispatchNotAllowed,
		},
		"create_job_error": {
			input: struct{ specs []execution.DispatchSpec }{specs: []execution.DispatchSpec{goodSpec}},
			mock:  struct{ createJobErr, publishErr, finalizeErr error }{createJobErr: errors.New("db")},
			err:   execution.ErrDispatchFailed,
		},
		"all_failed_bad_payload": {
			input:  struct{ specs []execution.DispatchSpec }{specs: []execution.DispatchSpec{badSpec}},
			output: string(model.JobStatusFailed),
			err:    execution.ErrDispatchFailed,
		},
		"all_failed_finalize_error": {
			input:  struct{ specs []execution.DispatchSpec }{specs: []execution.DispatchSpec{badSpec}},
			mock:   struct{ createJobErr, publishErr, finalizeErr error }{finalizeErr: errors.New("finalize")},
			output: string(model.JobStatusFailed),
			err:    execution.ErrDispatchFailed,
		},
		"partial_success": {
			input:  struct{ specs []execution.DispatchSpec }{specs: []execution.DispatchSpec{goodSpec, badSpec}},
			output: string(model.JobStatusPartial),
		},
		"default_requested_at": {
			input:  struct{ specs []execution.DispatchSpec }{specs: []execution.DispatchSpec{goodSpec}},
			output: string(model.JobStatusRunning),
		},
		"partial_finalize_error": {
			input: struct{ specs []execution.DispatchSpec }{specs: []execution.DispatchSpec{goodSpec, badSpec}},
			mock:  struct{ createJobErr, publishErr, finalizeErr error }{finalizeErr: errors.New("finalize")},
			err:   execution.ErrDispatchFailed,
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			uc, r, pub, project := newExecutionUC(t)
			if len(tc.input.specs) > 0 {
				r.EXPECT().CreateScheduledJob(context.Background(), mock.AnythingOfType("repository.CreateScheduledJobOptions")).Return(model.ScheduledJob{ID: "job-1"}, tc.mock.createJobErr).Once()
			}
			if tc.mock.createJobErr == nil && len(tc.input.specs) > 0 {
				for _, spec := range tc.input.specs {
					if _, ok := spec.Params["bad"]; ok {
						continue
					}
					project.EXPECT().Detail(context.Background(), testProjectID).Return(microservice.ProjectDetail{ID: testProjectID, Status: microservice.ProjectStatusActive, DomainTypeCode: "domain"}, nil).Once()
					r.EXPECT().CreateExternalTask(context.Background(), mock.AnythingOfType("repository.CreateExternalTaskOptions")).Return(model.ExternalTask{ID: "external-1"}, nil).Once()
					pub.EXPECT().PublishDispatch(context.Background(), mock.AnythingOfType("execution.PublishDispatchInput")).Return(tc.mock.publishErr).Once()
					if tc.mock.publishErr != nil {
						r.EXPECT().MarkExternalTaskFailed(context.Background(), mock.AnythingOfType("repository.MarkExternalTaskFailedOptions")).Return(nil).Once()
					} else {
						r.EXPECT().MarkExternalTaskPublished(context.Background(), mock.AnythingOfType("repository.MarkExternalTaskPublishedOptions")).Return(nil).Once()
					}
				}
				if name == "all_failed_bad_payload" || name == "all_failed_finalize_error" || name == "partial_success" || name == "partial_finalize_error" {
					r.EXPECT().FinalizeScheduledJob(context.Background(), mock.AnythingOfType("repository.FinalizeScheduledJobOptions")).Return(tc.mock.finalizeErr).Once()
				}
			}

			dispatchInput := execution.DispatchTargetInput{RequestedAt: now}
			if name == "default_requested_at" {
				dispatchInput = execution.DispatchTargetInput{}
			}
			output, err := uc.dispatchPrepared(context.Background(), ctx, tc.input.specs, dispatchInput)

			require.ErrorIs(t, err, tc.err)
			if tc.output != "" {
				require.Equal(t, tc.output, output.Status)
			}
		})
	}
}

func TestDispatchOneSpec(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	ctx := repo.DispatchContext{Source: executionSource(), Target: executionTarget()}
	spec := execution.DispatchSpec{Queue: execution.QueueName("q"), Action: execution.ActionNameFullFlow, Keyword: "vinfast", Params: map[string]interface{}{"keyword": "vinfast"}}

	tcs := map[string]struct {
		mock struct {
			projectErr error
			createErr  error
			publishErr error
			markErr    error
			failErr    error
		}
		err bool
	}{
		"project_error": {
			mock: struct {
				projectErr error
				createErr  error
				publishErr error
				markErr    error
				failErr    error
			}{projectErr: errors.New("project")},
			err: true,
		},
		"create_external_error": {
			mock: struct {
				projectErr error
				createErr  error
				publishErr error
				markErr    error
				failErr    error
			}{createErr: errors.New("db")},
			err: true,
		},
		"publish_error_mark_failed_error": {
			mock: struct {
				projectErr error
				createErr  error
				publishErr error
				markErr    error
				failErr    error
			}{publishErr: errors.New("publish"), failErr: errors.New("mark failed")},
			err: true,
		},
		"mark_published_error": {
			mock: struct {
				projectErr error
				createErr  error
				publishErr error
				markErr    error
				failErr    error
			}{markErr: errors.New("mark")},
			err: true,
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			uc, r, pub, project := newExecutionUC(t)
			project.EXPECT().Detail(context.Background(), testProjectID).Return(microservice.ProjectDetail{ID: testProjectID, Status: microservice.ProjectStatusActive, DomainTypeCode: "domain"}, tc.mock.projectErr).Once()
			if tc.mock.projectErr == nil {
				r.EXPECT().CreateExternalTask(context.Background(), mock.AnythingOfType("repository.CreateExternalTaskOptions")).Return(model.ExternalTask{ID: "external-1"}, tc.mock.createErr).Once()
			}
			if tc.mock.projectErr == nil && tc.mock.createErr == nil {
				pub.EXPECT().PublishDispatch(context.Background(), mock.AnythingOfType("execution.PublishDispatchInput")).Return(tc.mock.publishErr).Once()
				if tc.mock.publishErr != nil {
					r.EXPECT().MarkExternalTaskFailed(context.Background(), mock.AnythingOfType("repository.MarkExternalTaskFailedOptions")).Return(tc.mock.failErr).Once()
				} else {
					r.EXPECT().MarkExternalTaskPublished(context.Background(), mock.AnythingOfType("repository.MarkExternalTaskPublishedOptions")).Return(tc.mock.markErr).Once()
					if tc.mock.markErr != nil {
						r.EXPECT().MarkExternalTaskFailed(context.Background(), mock.AnythingOfType("repository.MarkExternalTaskFailedOptions")).Return(nil).Once()
					}
				}
			}

			_, err := uc.dispatchOneSpec(context.Background(), ctx, "job-1", spec, now)

			if tc.err {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestExecutionFlowHelpers(t *testing.T) {
	modeCrisis := model.CrawlModeCrisis
	modeSleep := model.CrawlModeSleep
	modeBad := model.CrawlMode("BAD")
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
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
			uc, _, _, _ := newExecutionUC(t)
			require.NoError(t, uc.validateScheduledDispatchContext(repo.DispatchContext{Source: executionSource(), Target: executionTarget()}))
			err := uc.validateCompletionInput(execution.HandleCompletionInput{TaskID: "", Status: ""})
			require.ErrorIs(t, err, execution.ErrInvalidCompletionInput)
			require.Equal(t, execution.CrisisModeMultiplier, mustMultiplier(t, uc, modeCrisis))
			require.Equal(t, execution.SleepModeMultiplier, mustMultiplier(t, uc, modeSleep))
			_, err = uc.getModeMultiplier(modeBad)
			require.Error(t, err)
			duration, err := uc.computeEffectiveInterval(model.DataSource{CrawlMode: &modeCrisis}, model.CrawlTarget{CrawlIntervalMinutes: 1})
			require.NoError(t, err)
			require.Equal(t, time.Minute, duration)
			_, err = uc.computeEffectiveInterval(model.DataSource{}, model.CrawlTarget{CrawlIntervalMinutes: 1})
			require.Error(t, err)
			_, err = uc.computeEffectiveInterval(model.DataSource{CrawlMode: &modeCrisis}, model.CrawlTarget{})
			require.Error(t, err)
			require.Equal(t, int64Ptr(2), uc.parseInt64(json.Number("2")))
			payload, err := uc.buildRequestPayload("task-1", execution.ActionNameFullFlow, map[string]interface{}{"keyword": "vinfast"}, now)
			require.NoError(t, err)
			require.Contains(t, string(payload), "task-1")
			require.NotEmpty(t, uc.buildJobPayload([]execution.DispatchSpec{{Queue: "q", Action: "a", Keyword: "k", Params: map[string]interface{}{"x": 1}}}, execution.DispatchTargetInput{TriggerType: model.TriggerTypeManual, ScheduledFor: now, RequestedAt: now, CronExpr: "*/5 * * * *"}))
			require.Equal(t, "dispatch fan-out encountered 3 failure(s): a; b", uc.summarizeDispatchFailures([]string{"a", "b", "a"}, 3))
			require.Equal(t, "failed to publish 2 external task(s)", uc.summarizeDispatchFailures(nil, 2))
		})
	}
}

func TestHandleCompletion(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	errDB := errors.New("db")
	task := model.ExternalTask{ID: "external-1", SourceID: testSourceID, ProjectID: testProjectID, DomainTypeCode: "domain", TaskID: "task-1", Platform: "tiktok", TaskType: "full_flow", Status: model.JobStatusRunning}
	failed := task
	failed.Status = model.JobStatusFailed
	failed.CompletedAt = &now
	cancelled := task
	cancelled.Status = model.JobStatusCancelled
	cancelled.CompletedAt = &now
	type mockCompletion struct {
		getCalled      bool
		getOutput      repo.CompletionContext
		getErr         error
		hasCalled      bool
		hasOutput      bool
		hasErr         error
		errorCalled    bool
		errorInput     repo.CompleteTaskErrorOptions
		errorErr       error
		successCalled  bool
		successOutput  model.RawBatch
		successErr     error
		minioOK        bool
		parseSupported bool
		parseCalled    bool
	}

	tcs := map[string]struct {
		input  execution.HandleCompletionInput
		mock   mockCompletion
		output struct{}
		err    error
	}{
		"invalid": {input: execution.HandleCompletionInput{}, err: execution.ErrInvalidCompletionInput},
		"not_found": {
			input: execution.HandleCompletionInput{TaskID: "task-1", Status: "error"},
			mock:  mockCompletion{getCalled: true, getErr: repo.ErrExternalTaskNotFound},
			err:   execution.ErrCompletionTaskNotFound,
		},
		"get_context_error": {
			input: execution.HandleCompletionInput{TaskID: "task-1", Status: "error"},
			mock:  mockCompletion{getCalled: true, getErr: errDB},
			err:   errDB,
		},
		"cancelled_duplicate": {
			input: execution.HandleCompletionInput{TaskID: "task-1", Status: "error"},
			mock:  mockCompletion{getCalled: true, getOutput: repo.CompletionContext{ExternalTask: cancelled}},
		},
		"failed_duplicate": {
			input: execution.HandleCompletionInput{TaskID: "task-1", Status: "error"},
			mock:  mockCompletion{getCalled: true, getOutput: repo.CompletionContext{ExternalTask: failed}},
		},
		"error_status": {
			input: execution.HandleCompletionInput{TaskID: "task-1", Status: "error", Error: " boom ", CompletedAt: now.Format(time.RFC3339)},
			mock:  mockCompletion{getCalled: true, getOutput: repo.CompletionContext{ExternalTask: task}, errorCalled: true, errorInput: repo.CompleteTaskErrorOptions{CompletionContext: repo.CompletionContext{ExternalTask: task}, ErrorMessage: "boom", CompletedAt: now}},
		},
		"error_default_message": {
			input: execution.HandleCompletionInput{TaskID: "task-1", Status: "error", CompletedAt: now.Format(time.RFC3339)},
			mock:  mockCompletion{getCalled: true, getOutput: repo.CompletionContext{ExternalTask: task}, errorCalled: true, errorInput: repo.CompleteTaskErrorOptions{CompletionContext: repo.CompletionContext{ExternalTask: task}, ErrorMessage: "crawler returned error completion without error message", CompletedAt: now}},
		},
		"error_complete_failed": {
			input: execution.HandleCompletionInput{TaskID: "task-1", Status: "error", Error: "boom", CompletedAt: now.Format(time.RFC3339)},
			mock:  mockCompletion{getCalled: true, getOutput: repo.CompletionContext{ExternalTask: task}, errorCalled: true, errorErr: errDB},
			err:   errDB,
		},
		"success_duplicate_batch": {
			input: execution.HandleCompletionInput{TaskID: "task-1", Status: "success", StorageBucket: "bucket", StoragePath: "path", BatchID: "batch-1"},
			mock:  mockCompletion{getCalled: true, getOutput: repo.CompletionContext{ExternalTask: task}, hasCalled: true, hasOutput: true},
		},
		"success_has_raw_batch_error": {
			input: execution.HandleCompletionInput{TaskID: "task-1", Status: "success", StorageBucket: "bucket", StoragePath: "path", BatchID: "batch-1"},
			mock:  mockCompletion{getCalled: true, getOutput: repo.CompletionContext{ExternalTask: task}, hasCalled: true, hasErr: errDB},
			err:   errDB,
		},
		"success_verify_failed": {
			input: execution.HandleCompletionInput{TaskID: "task-1", Status: "success", StorageBucket: "bucket", StoragePath: "path", BatchID: "batch-1"},
			mock:  mockCompletion{getCalled: true, getOutput: repo.CompletionContext{ExternalTask: task}, hasCalled: true, errorCalled: true},
		},
		"success_verify_failed_complete_error": {
			input: execution.HandleCompletionInput{TaskID: "task-1", Status: "success", StorageBucket: "bucket", StoragePath: "path", BatchID: "batch-1"},
			mock:  mockCompletion{getCalled: true, getOutput: repo.CompletionContext{ExternalTask: task}, hasCalled: true, errorCalled: true, errorErr: errDB},
			err:   errDB,
		},
		"success_metadata_marshal_error": {
			input: execution.HandleCompletionInput{TaskID: "task-1", Status: "success", StorageBucket: "bucket", StoragePath: "path", BatchID: "batch-1", Metadata: map[string]interface{}{"bad": func() {}}},
			mock:  mockCompletion{getCalled: true, getOutput: repo.CompletionContext{ExternalTask: task}, hasCalled: true, minioOK: true},
			err:   execution.ErrInvalidCompletionInput,
		},
		"success_complete_already_exists": {
			input: execution.HandleCompletionInput{TaskID: "task-1", Status: "success", StorageBucket: "bucket", StoragePath: "path", BatchID: "batch-1"},
			mock:  mockCompletion{getCalled: true, getOutput: repo.CompletionContext{ExternalTask: task}, hasCalled: true, minioOK: true, successCalled: true, successErr: repo.ErrRawBatchAlreadyExists},
		},
		"success_complete_error": {
			input: execution.HandleCompletionInput{TaskID: "task-1", Status: "success", StorageBucket: "bucket", StoragePath: "path", BatchID: "batch-1"},
			mock:  mockCompletion{getCalled: true, getOutput: repo.CompletionContext{ExternalTask: task}, hasCalled: true, minioOK: true, successCalled: true, successErr: errDB},
			err:   errDB,
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			uc, r, _, _ := newExecutionUC(t)
			if tc.mock.getCalled {
				r.EXPECT().GetCompletionContext(context.Background(), "task-1").Return(tc.mock.getOutput, tc.mock.getErr)
			}
			if tc.mock.hasCalled {
				r.EXPECT().HasRawBatch(context.Background(), testSourceID, "batch-1").Return(tc.mock.hasOutput, tc.mock.hasErr)
			}
			if tc.mock.minioOK {
				uc.minio = fakeMinIO{exists: true, info: &sharedMinio.FileInfo{Size: 123}}
			}
			if tc.mock.errorCalled {
				r.EXPECT().CompleteTaskError(context.Background(), mock.MatchedBy(func(opt repo.CompleteTaskErrorOptions) bool {
					if opt.CompletionContext.ExternalTask.ID != task.ID {
						return false
					}
					if tc.mock.errorInput.ErrorMessage != "" && opt.ErrorMessage != tc.mock.errorInput.ErrorMessage {
						return false
					}
					return true
				})).Return(tc.mock.errorErr)
			}
			if tc.mock.successCalled {
				r.EXPECT().CompleteTaskSuccess(context.Background(), mock.AnythingOfType("repository.CompleteTaskSuccessOptions")).Return(tc.mock.successOutput, tc.mock.successErr)
			}

			err := uc.HandleCompletion(context.Background(), tc.input)

			require.ErrorIs(t, err, tc.err)
		})
	}
}

func TestHandleCompletionSuccess(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	itemCount := 3
	task := model.ExternalTask{ID: "external-1", SourceID: testSourceID, ProjectID: testProjectID, DomainTypeCode: "domain", TaskID: "task-1", Platform: "tiktok", TaskType: "full_flow", Status: model.JobStatusRunning}
	rawBatch := model.RawBatch{ID: "raw-1", SourceID: testSourceID, ProjectID: testProjectID, DomainTypeCode: "domain", ExternalTaskID: "external-1", BatchID: "batch-1", StorageBucket: "bucket", StoragePath: "path", RawMetadata: []byte(`{"size_bytes":123}`)}
	tcs := map[string]struct {
		input execution.HandleCompletionInput
		mock  struct {
			parseErr error
		}
		output struct{}
		err    error
	}{
		"success_with_parse": {
			input: execution.HandleCompletionInput{TaskID: "task-1", Status: "success", CompletedAt: now.Format(time.RFC3339), StorageBucket: "bucket", StoragePath: "path", BatchID: "batch-1", ItemCount: &itemCount, Metadata: map[string]interface{}{"size_bytes": float64(123)}},
		},
		"success_parse_error_ignored": {
			input: execution.HandleCompletionInput{TaskID: "task-1", Status: "success", CompletedAt: now.Format(time.RFC3339), StorageBucket: "bucket", StoragePath: "path", BatchID: "batch-1", ItemCount: &itemCount, Metadata: map[string]interface{}{"size_bytes": float64(123)}},
			mock: struct{ parseErr error }{
				parseErr: errors.New("parse"),
			},
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			uc, r, _, _ := newExecutionUC(t)
			uc.minio = fakeMinIO{exists: true, info: &sharedMinio.FileInfo{Size: 123}}
			parser := uap.NewMockUseCase(t)
			uc.parser = parser
			r.EXPECT().GetCompletionContext(context.Background(), "task-1").Return(repo.CompletionContext{ExternalTask: task}, nil)
			r.EXPECT().HasRawBatch(context.Background(), testSourceID, "batch-1").Return(false, nil)
			r.EXPECT().CompleteTaskSuccess(context.Background(), mock.MatchedBy(func(opt repo.CompleteTaskSuccessOptions) bool {
				return opt.CompletionContext.ExternalTask.ID == task.ID && opt.BatchID == "batch-1" && opt.SizeBytes != nil && *opt.SizeBytes == 123
			})).Return(rawBatch, nil)
			parser.EXPECT().SupportsParse("tiktok", "full_flow").Return(true)
			parser.EXPECT().ParseAndStoreRawBatch(context.Background(), mock.MatchedBy(func(input uap.ParseAndStoreRawBatchInput) bool {
				return input.RawBatchID == "raw-1" && input.ExternalTaskID == "external-1"
			})).Return(tc.mock.parseErr)

			err := uc.HandleCompletion(context.Background(), tc.input)

			require.ErrorIs(t, err, tc.err)
		})
	}
}

func TestShouldParseUAP(t *testing.T) {
	tcs := map[string]struct {
		input  model.ExternalTask
		mock   bool
		output bool
		err    error
	}{
		"nil_parser":  {input: model.ExternalTask{Platform: "tiktok", TaskType: "full_flow"}, output: false},
		"supported":   {input: model.ExternalTask{Platform: "tiktok", TaskType: "full_flow"}, mock: true, output: true},
		"unsupported": {input: model.ExternalTask{Platform: "x", TaskType: "y"}, mock: false, output: false},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			uc, _, _, _ := newExecutionUC(t)
			if name != "nil_parser" {
				parser := uap.NewMockUseCase(t)
				parser.EXPECT().SupportsParse(tc.input.Platform, tc.input.TaskType).Return(tc.mock)
				uc.parser = parser
			}

			output := uc.shouldParseUAP(tc.input)

			require.Equal(t, tc.output, output)
		})
	}
}

func TestDispatchDueTargets(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	baseDue := repo.DueTarget{Source: executionSource(), Target: executionTarget()}
	baseCtx := repo.DispatchContext{Source: executionSource(), Target: executionTarget()}

	type mockDue struct {
		dueTargets []repo.DueTarget
		listErr    error
		claim      bool
		claimErr   error
		afterCtx   repo.DispatchContext
		afterErr   error
		createErr  error
		publishErr error
	}

	tcs := map[string]struct {
		input  execution.DispatchDueTargetsInput
		mock   mockDue
		output execution.DispatchDueTargetsOutput
		err    error
	}{
		"success": {
			input:  execution.DispatchDueTargetsInput{Now: now, Limit: 1, CronExpr: "* * * * *"},
			mock:   mockDue{dueTargets: []repo.DueTarget{baseDue}, claim: true, afterCtx: baseCtx},
			output: execution.DispatchDueTargetsOutput{DueCount: 1, ClaimedCount: 1, DispatchedCount: 1},
		},
		"default_now_limit_empty": {
			input:  execution.DispatchDueTargetsInput{},
			output: execution.DispatchDueTargetsOutput{},
		},
		"list_error": {
			input: execution.DispatchDueTargetsInput{Now: now, Limit: 1},
			mock:  mockDue{listErr: repo.ErrListDueTargets},
			err:   execution.ErrDispatchFailed,
		},
		"invalid_due_context": {
			input: execution.DispatchDueTargetsInput{Now: now, Limit: 1},
			mock: mockDue{dueTargets: []repo.DueTarget{func() repo.DueTarget {
				due := baseDue
				due.Source.Status = model.SourceStatusPaused
				return due
			}()}},
			output: execution.DispatchDueTargetsOutput{DueCount: 1, FailedCount: 1},
		},
		"unsupported_dispatch_mapping": {
			input: execution.DispatchDueTargetsInput{Now: now, Limit: 1},
			mock: mockDue{dueTargets: []repo.DueTarget{func() repo.DueTarget {
				due := baseDue
				due.Source.SourceType = model.SourceTypeWebhook
				return due
			}()}},
			output: execution.DispatchDueTargetsOutput{DueCount: 1, FailedCount: 1},
		},
		"interval_error": {
			input: execution.DispatchDueTargetsInput{Now: now, Limit: 1},
			mock: mockDue{dueTargets: []repo.DueTarget{func() repo.DueTarget {
				due := baseDue
				due.Target.CrawlIntervalMinutes = 0
				return due
			}()}},
			output: execution.DispatchDueTargetsOutput{DueCount: 1, FailedCount: 1},
		},
		"claim_error": {
			input:  execution.DispatchDueTargetsInput{Now: now, Limit: 1},
			mock:   mockDue{dueTargets: []repo.DueTarget{baseDue}, claimErr: errors.New("claim")},
			output: execution.DispatchDueTargetsOutput{DueCount: 1, FailedCount: 1},
		},
		"claim_race": {
			input:  execution.DispatchDueTargetsInput{Now: now, Limit: 1},
			mock:   mockDue{dueTargets: []repo.DueTarget{baseDue}, claim: false},
			output: execution.DispatchDueTargetsOutput{DueCount: 1, SkippedRaceCount: 1},
		},
		"after_claim_context_error": {
			input:  execution.DispatchDueTargetsInput{Now: now, Limit: 1},
			mock:   mockDue{dueTargets: []repo.DueTarget{baseDue}, claim: true, afterErr: repo.ErrTargetNotFound},
			output: execution.DispatchDueTargetsOutput{DueCount: 1, ClaimedCount: 1, SkippedRaceCount: 1},
		},
		"after_claim_context_invalid": {
			input: execution.DispatchDueTargetsInput{Now: now, Limit: 1},
			mock: mockDue{dueTargets: []repo.DueTarget{baseDue}, claim: true, afterCtx: func() repo.DispatchContext {
				ctx := baseCtx
				ctx.Target.IsActive = false
				return ctx
			}()},
			output: execution.DispatchDueTargetsOutput{DueCount: 1, ClaimedCount: 1, SkippedRaceCount: 1},
		},
		"create_job_conflict_skipped": {
			input:  execution.DispatchDueTargetsInput{Now: now, Limit: 1},
			mock:   mockDue{dueTargets: []repo.DueTarget{baseDue}, claim: true, afterCtx: baseCtx, createErr: repo.ErrDispatchConflict},
			output: execution.DispatchDueTargetsOutput{DueCount: 1, ClaimedCount: 1, SkippedRaceCount: 1},
		},
		"dispatch_failed": {
			input:  execution.DispatchDueTargetsInput{Now: now, Limit: 1},
			mock:   mockDue{dueTargets: []repo.DueTarget{baseDue}, claim: true, afterCtx: baseCtx, publishErr: errors.New("publish")},
			output: execution.DispatchDueTargetsOutput{DueCount: 1, ClaimedCount: 1, FailedCount: 1},
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			uc, r, pub, project := newExecutionUC(t)
			if tc.mock.listErr != nil {
				r.EXPECT().ListDueTargets(context.Background(), now, 1).Return(nil, tc.mock.listErr)
			} else {
				expectedNow := tc.input.Now
				if expectedNow.IsZero() {
					expectedNow = now
				}
				expectedLimit := tc.input.Limit
				if expectedLimit <= 0 {
					expectedLimit = 1
				}
				r.EXPECT().ListDueTargets(context.Background(), expectedNow, expectedLimit).Return(tc.mock.dueTargets, nil)
				if len(tc.mock.dueTargets) > 0 && tc.mock.dueTargets[0].Source.Status == model.SourceStatusActive && tc.mock.dueTargets[0].Source.SourceType != model.SourceTypeWebhook && tc.mock.dueTargets[0].Target.CrawlIntervalMinutes > 0 {
					r.EXPECT().ClaimTarget(context.Background(), mock.AnythingOfType("repository.ClaimTargetOptions")).Return(tc.mock.claim, tc.mock.claimErr)
				}
				if tc.mock.claim && tc.mock.claimErr == nil {
					r.EXPECT().GetDispatchContext(context.Background(), testSourceID, testTargetID).Return(tc.mock.afterCtx, tc.mock.afterErr)
					if tc.mock.afterErr != nil || !tc.mock.afterCtx.Target.IsActive {
						r.EXPECT().ReleaseClaimTarget(context.Background(), repo.ReleaseClaimTargetOptions{SourceID: testSourceID, TargetID: testTargetID}).Return(nil)
					}
					if tc.mock.afterErr == nil && tc.mock.afterCtx.Target.IsActive {
						r.EXPECT().CreateScheduledJob(context.Background(), mock.AnythingOfType("repository.CreateScheduledJobOptions")).Return(model.ScheduledJob{ID: "job-1"}, tc.mock.createErr)
						if tc.mock.createErr == nil {
							project.EXPECT().Detail(context.Background(), testProjectID).Return(microservice.ProjectDetail{ID: testProjectID, Status: microservice.ProjectStatusActive, DomainTypeCode: "domain"}, nil)
							r.EXPECT().CreateExternalTask(context.Background(), mock.AnythingOfType("repository.CreateExternalTaskOptions")).Return(model.ExternalTask{ID: "external-1"}, nil)
							pub.EXPECT().PublishDispatch(context.Background(), mock.AnythingOfType("execution.PublishDispatchInput")).Return(tc.mock.publishErr)
							if tc.mock.publishErr != nil {
								r.EXPECT().MarkExternalTaskFailed(context.Background(), mock.AnythingOfType("repository.MarkExternalTaskFailedOptions")).Return(nil)
								r.EXPECT().FinalizeScheduledJob(context.Background(), mock.AnythingOfType("repository.FinalizeScheduledJobOptions")).Return(nil)
							} else {
								r.EXPECT().MarkExternalTaskPublished(context.Background(), mock.AnythingOfType("repository.MarkExternalTaskPublishedOptions")).Return(nil)
							}
						}
					}
				}
			}

			output, err := uc.DispatchDueTargets(context.Background(), tc.input)

			require.ErrorIs(t, err, tc.err)
			if tc.err == nil {
				require.Equal(t, tc.output, output)
			}
		})
	}
}

func TestCancelProjectRuntime(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	tcs := map[string]struct {
		input  execution.CancelProjectRuntimeInput
		mock   error
		output struct{}
		err    error
	}{
		"success":    {input: execution.CancelProjectRuntimeInput{ProjectID: " " + testProjectID + " ", Reason: " reason ", CanceledAt: now}},
		"invalid":    {err: execution.ErrCancelRuntimeFailed},
		"repo_error": {input: execution.CancelProjectRuntimeInput{ProjectID: testProjectID}, mock: repo.ErrCancelRuntime, err: execution.ErrCancelRuntimeFailed},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			uc, r, _, _ := newExecutionUC(t)
			if tc.input.ProjectID != "" {
				r.EXPECT().CancelProjectRuntime(context.Background(), repo.CancelProjectRuntimeOptions{ProjectID: testProjectID, Reason: strings.TrimSpace(tc.input.Reason), CanceledAt: tc.input.CanceledAt}).Return(tc.mock)
			}

			err := uc.CancelProjectRuntime(context.Background(), tc.input)

			require.ErrorIs(t, err, tc.err)
		})
	}
}

func TestResolveProjectDomainTypeCode(t *testing.T) {
	tcs := map[string]struct {
		input  string
		mock   microservice.ProjectDetail
		output string
		err    error
	}{
		"success":     {input: testProjectID, mock: microservice.ProjectDetail{ID: testProjectID, Status: microservice.ProjectStatusActive, DomainTypeCode: "domain"}, output: "domain"},
		"default":     {input: testProjectID, mock: microservice.ProjectDetail{ID: testProjectID, Status: microservice.ProjectStatusActive}, output: "generic"},
		"archived":    {input: testProjectID, mock: microservice.ProjectDetail{ID: testProjectID, Status: microservice.ProjectStatusArchived}, err: execution.ErrDispatchNotAllowed},
		"nil_project": {err: errAny},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			uc, _, _, project := newExecutionUC(t)
			if name == "nil_project" {
				uc.project = nil
			} else {
				project.EXPECT().Detail(context.Background(), testProjectID).Return(tc.mock, nil)
			}

			output, err := uc.resolveProjectDomainTypeCode(context.Background(), tc.input)

			if tc.err == errAny {
				require.Error(t, err)
			} else {
				require.ErrorIs(t, err, tc.err)
			}
			require.Equal(t, tc.output, output)
		})
	}
}

var errAny = errors.New("any error")

func mustMultiplier(t *testing.T, uc *implUseCase, mode model.CrawlMode) float64 {
	t.Helper()
	value, err := uc.getModeMultiplier(mode)
	require.NoError(t, err)
	return value
}

func int64Ptr(v int64) *int64 {
	return &v
}

type fakeMinIO struct {
	exists    bool
	info      *sharedMinio.FileInfo
	err       error
	existsErr error
	statErr   error
}

func (f fakeMinIO) Connect(context.Context) error                                  { return nil }
func (f fakeMinIO) ConnectWithRetry(context.Context, int) error                    { return nil }
func (f fakeMinIO) HealthCheck(context.Context) error                              { return nil }
func (f fakeMinIO) Close() error                                                   { return nil }
func (f fakeMinIO) CreateBucket(context.Context, string) error                     { return nil }
func (f fakeMinIO) DeleteBucket(context.Context, string) error                     { return nil }
func (f fakeMinIO) BucketExists(context.Context, string) (bool, error)             { return true, nil }
func (f fakeMinIO) ListBuckets(context.Context) ([]*sharedMinio.BucketInfo, error) { return nil, nil }
func (f fakeMinIO) UploadFile(context.Context, *sharedMinio.UploadRequest) (*sharedMinio.FileInfo, error) {
	return nil, nil
}
func (f fakeMinIO) GetPresignedUploadURL(context.Context, *sharedMinio.PresignedURLRequest) (*sharedMinio.PresignedURLResponse, error) {
	return nil, nil
}
func (f fakeMinIO) DownloadFile(context.Context, *sharedMinio.DownloadRequest) (io.ReadCloser, *sharedMinio.DownloadHeaders, error) {
	return nil, nil, nil
}
func (f fakeMinIO) StreamFile(context.Context, *sharedMinio.DownloadRequest) (io.ReadCloser, *sharedMinio.DownloadHeaders, error) {
	return nil, nil, nil
}
func (f fakeMinIO) GetPresignedDownloadURL(context.Context, *sharedMinio.PresignedURLRequest) (*sharedMinio.PresignedURLResponse, error) {
	return nil, nil
}
func (f fakeMinIO) GetFileInfo(context.Context, string, string) (*sharedMinio.FileInfo, error) {
	if f.statErr != nil {
		return nil, f.statErr
	}
	return f.info, f.err
}
func (f fakeMinIO) DeleteFile(context.Context, string, string) error               { return nil }
func (f fakeMinIO) CopyFile(context.Context, string, string, string, string) error { return nil }
func (f fakeMinIO) MoveFile(context.Context, string, string, string, string) error { return nil }
func (f fakeMinIO) FileExists(context.Context, string, string) (bool, error) {
	if f.existsErr != nil {
		return false, f.existsErr
	}
	return f.exists, f.err
}
func (f fakeMinIO) ListFiles(context.Context, *sharedMinio.ListRequest) (*sharedMinio.ListResponse, error) {
	return nil, nil
}
func (f fakeMinIO) UpdateMetadata(context.Context, string, string, map[string]string) error {
	return nil
}
func (f fakeMinIO) GetMetadata(context.Context, string, string) (map[string]string, error) {
	return nil, nil
}
func (f fakeMinIO) UploadAsync(context.Context, *sharedMinio.UploadRequest) (string, error) {
	return "", nil
}
func (f fakeMinIO) GetUploadStatus(string) (*sharedMinio.UploadProgress, error) { return nil, nil }
func (f fakeMinIO) WaitForUpload(string, time.Duration) (*sharedMinio.AsyncUploadResult, error) {
	return nil, nil
}
func (f fakeMinIO) CancelUpload(string) error { return nil }
