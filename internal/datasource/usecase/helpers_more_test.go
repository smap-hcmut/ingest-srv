package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"ingest-srv/internal/datasource"
	repo "ingest-srv/internal/datasource/repository"
	"ingest-srv/internal/execution"
	"ingest-srv/internal/model"
	"ingest-srv/pkg/microservice"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestValidationHelpers(t *testing.T) {
	uc := &implUseCase{}

	tcs := map[string]struct {
		input func() error
		mock  struct{}
		err   error
	}{
		"valid crawl category":    {input: func() error { return uc.validateSourceCategory(string(model.SourceCategoryCrawl)) }},
		"valid passive category":  {input: func() error { return uc.validateSourceCategory(string(model.SourceCategoryPassive)) }},
		"invalid category":        {input: func() error { return uc.validateSourceCategory("bad") }, err: datasource.ErrInvalidCategory},
		"valid manual trigger":    {input: func() error { return uc.validateTriggerType(string(model.TriggerTypeManual)) }},
		"valid scheduled trigger": {input: func() error { return uc.validateTriggerType(string(model.TriggerTypeScheduled)) }},
		"valid project trigger":   {input: func() error { return uc.validateTriggerType(string(model.TriggerTypeProjectEvent)) }},
		"valid crisis trigger":    {input: func() error { return uc.validateTriggerType(string(model.TriggerTypeCrisisEvent)) }},
		"valid webhook trigger":   {input: func() error { return uc.validateTriggerType(string(model.TriggerTypeWebhookPush)) }},
		"invalid trigger":         {input: func() error { return uc.validateTriggerType("bad") }, err: datasource.ErrCrawlModeNotAllowed},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			err := tc.input()
			require.ErrorIs(t, err, tc.err)
		})
	}
}

func TestTargetMutationHelpers(t *testing.T) {
	uc := &implUseCase{}
	interval := 20

	tcs := map[string]struct {
		input  func() bool
		mock   struct{}
		output bool
		err    error
	}{
		"string slices equal": {
			input:  func() bool { return uc.areStringSlicesEqual([]string{" a "}, []string{"a"}) },
			output: true,
		},
		"string slices length mismatch": {
			input: func() bool { return uc.areStringSlicesEqual([]string{"a"}, []string{"a", "b"}) },
		},
		"string slices value mismatch": {
			input: func() bool { return uc.areStringSlicesEqual([]string{"a"}, []string{"b"}) },
		},
		"material values changed": {
			input: func() bool {
				return uc.hasMaterialTargetChange(testCrawlTarget(true), []string{"other"}, datasource.UpdateTargetInput{Values: []string{"other"}})
			},
			output: true,
		},
		"material interval changed": {
			input: func() bool {
				return uc.hasMaterialTargetChange(testCrawlTarget(true), []string{"vinfast"}, datasource.UpdateTargetInput{CrawlIntervalMinutes: &interval})
			},
			output: true,
		},
		"material meta changed": {
			input: func() bool {
				return uc.hasMaterialTargetChange(testCrawlTarget(true), []string{"vinfast"}, datasource.UpdateTargetInput{PlatformMeta: []byte(`{"a":1}`)})
			},
			output: true,
		},
		"no material change": {
			input: func() bool {
				current := testCrawlTarget(true)
				current.PlatformMeta = []byte(`{"a":1}`)
				return uc.hasMaterialTargetChange(current, []string{"vinfast"}, datasource.UpdateTargetInput{PlatformMeta: []byte(`{"a":1}`)})
			},
		},
		"source dryrun running": {
			input: func() bool {
				source := testSource(model.SourceStatusReady)
				source.DryrunStatus = model.DryrunStatusRunning
				return uc.isDatasourceDryrunRunning(source)
			},
			output: true,
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, tc.output, tc.input())
		})
	}
}

func TestDryrunGuardHelpers(t *testing.T) {
	errRepo := errors.New("repo")

	tcs := map[string]struct {
		mock func(*repo.MockRepository)
		run  func(*implUseCase) error
		err  error
	}{
		"ensure source dryrun not running": {
			run: func(uc *implUseCase) error {
				source := testSource(model.SourceStatusReady)
				source.DryrunStatus = model.DryrunStatusRunning
				return uc.ensureDatasourceDryrunNotRunning(source)
			},
			err: datasource.ErrSourceDryrunRunning,
		},
		"ensure target mutation not running": {
			run: func(uc *implUseCase) error {
				source := testSource(model.SourceStatusReady)
				source.DryrunStatus = model.DryrunStatusRunning
				return uc.ensureDatasourceTargetMutationNotRunning(source)
			},
			err: datasource.ErrTargetDryrunRunning,
		},
		"latest dryrun repo error": {
			mock: func(r *repo.MockRepository) {
				r.EXPECT().GetLatestDryrunByTarget(mock.Anything, testTargetID).Return(model.DryrunResult{}, errRepo).Once()
			},
			run: func(uc *implUseCase) error {
				_, err := uc.getLatestDryrunStatusByTarget(context.Background(), testTargetID)
				return err
			},
			err: datasource.ErrUpdateFailed,
		},
		"target dryrun running": {
			mock: func(r *repo.MockRepository) {
				r.EXPECT().GetLatestDryrunByTarget(mock.Anything, testTargetID).Return(model.DryrunResult{ID: "dryrun-1", Status: model.DryrunStatusRunning}, nil).Once()
			},
			run: func(uc *implUseCase) error {
				return uc.ensureTargetDryrunNotRunning(context.Background(), testTargetID)
			},
			err: datasource.ErrTargetDryrunRunning,
		},
		"datasource target list error": {
			mock: func(r *repo.MockRepository) {
				r.EXPECT().ListTargets(mock.Anything, repo.ListTargetsOptions{DataSourceID: testSourceID}).Return(nil, errRepo).Once()
			},
			run: func(uc *implUseCase) error {
				return uc.ensureDatasourceTargetsDryrunNotRunning(context.Background(), testSourceID)
			},
			err: datasource.ErrUpdateFailed,
		},
		"datasource target status error": {
			mock: func(r *repo.MockRepository) {
				r.EXPECT().ListTargets(mock.Anything, repo.ListTargetsOptions{DataSourceID: testSourceID}).Return([]model.CrawlTarget{testCrawlTarget(true)}, nil).Once()
				r.EXPECT().GetLatestDryrunByTarget(mock.Anything, testTargetID).Return(model.DryrunResult{}, errRepo).Once()
			},
			run: func(uc *implUseCase) error {
				return uc.ensureDatasourceTargetsDryrunNotRunning(context.Background(), testSourceID)
			},
			err: datasource.ErrUpdateFailed,
		},
		"datasource target running": {
			mock: func(r *repo.MockRepository) {
				r.EXPECT().ListTargets(mock.Anything, repo.ListTargetsOptions{DataSourceID: testSourceID}).Return([]model.CrawlTarget{testCrawlTarget(true)}, nil).Once()
				r.EXPECT().GetLatestDryrunByTarget(mock.Anything, testTargetID).Return(model.DryrunResult{ID: "dryrun-1", Status: model.DryrunStatusRunning}, nil).Once()
			},
			run: func(uc *implUseCase) error {
				return uc.ensureDatasourceTargetsDryrunNotRunning(context.Background(), testSourceID)
			},
			err: datasource.ErrSourceDryrunRunning,
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			uc, r, _ := newDatasourceUC(t)
			if tc.mock != nil {
				tc.mock(r)
			}

			err := tc.run(uc)
			require.ErrorIs(t, err, tc.err)
		})
	}
}

func TestEnsureProjectAvailableForDatasourceCreate(t *testing.T) {
	tcs := map[string]struct {
		mock func(*microservice.MockProjectUseCase)
		nil  bool
		err  error
	}{
		"nil project client": {nil: true, err: datasource.ErrCreateFailed},
		"project not found": {
			mock: func(p *microservice.MockProjectUseCase) {
				p.EXPECT().Detail(mock.Anything, testProjectID).Return(microservice.ProjectDetail{}, microservice.ErrBadRequest).Once()
			},
			err: datasource.ErrProjectNotFound,
		},
		"project unauthorized": {
			mock: func(p *microservice.MockProjectUseCase) {
				p.EXPECT().Detail(mock.Anything, testProjectID).Return(microservice.ProjectDetail{}, microservice.ErrUnauthorized).Once()
			},
			err: datasource.ErrCreateFailed,
		},
		"project error": {
			mock: func(p *microservice.MockProjectUseCase) {
				p.EXPECT().Detail(mock.Anything, testProjectID).Return(microservice.ProjectDetail{}, errors.New("project")).Once()
			},
			err: datasource.ErrCreateFailed,
		},
		"project archived": {
			mock: func(p *microservice.MockProjectUseCase) {
				p.EXPECT().Detail(mock.Anything, testProjectID).Return(microservice.ProjectDetail{ID: testProjectID, Status: microservice.ProjectStatusArchived}, nil).Once()
			},
			err: datasource.ErrProjectArchived,
		},
		"success": {
			mock: func(p *microservice.MockProjectUseCase) {
				p.EXPECT().Detail(mock.Anything, testProjectID).Return(microservice.ProjectDetail{ID: testProjectID, Status: microservice.ProjectStatusActive}, nil).Once()
			},
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			uc, _, project := newDatasourceUC(t)
			if tc.nil {
				uc.project = nil
			} else if tc.mock != nil {
				tc.mock(project)
			}

			err := uc.ensureProjectAvailableForDatasourceCreate(context.Background(), testProjectID)
			require.ErrorIs(t, err, tc.err)
		})
	}
}

func TestEnsureCanRemoveActiveTarget(t *testing.T) {
	errRepo := errors.New("repo")

	tcs := map[string]struct {
		input struct {
			active bool
		}
		mock func(*repo.MockRepository)
		err  error
	}{
		"inactive target": {},
		"detail error": {
			input: struct{ active bool }{active: true},
			mock: func(r *repo.MockRepository) {
				r.EXPECT().DetailDataSource(mock.Anything, testSourceID).Return(model.DataSource{}, errRepo).Once()
			},
			err: datasource.ErrTargetUpdateFailed,
		},
		"source not found": {
			input: struct{ active bool }{active: true},
			mock: func(r *repo.MockRepository) {
				r.EXPECT().DetailDataSource(mock.Anything, testSourceID).Return(model.DataSource{}, nil).Once()
			},
			err: datasource.ErrNotFound,
		},
		"source not active": {
			input: struct{ active bool }{active: true},
			mock: func(r *repo.MockRepository) {
				r.EXPECT().DetailDataSource(mock.Anything, testSourceID).Return(testSource(model.SourceStatusReady), nil).Once()
			},
		},
		"count active error": {
			input: struct{ active bool }{active: true},
			mock: func(r *repo.MockRepository) {
				r.EXPECT().DetailDataSource(mock.Anything, testSourceID).Return(testSource(model.SourceStatusActive), nil).Once()
				r.EXPECT().CountActiveTargets(mock.Anything, testSourceID).Return(int64(0), errRepo).Once()
			},
			err: datasource.ErrTargetUpdateFailed,
		},
		"last active target": {
			input: struct{ active bool }{active: true},
			mock: func(r *repo.MockRepository) {
				r.EXPECT().DetailDataSource(mock.Anything, testSourceID).Return(testSource(model.SourceStatusActive), nil).Once()
				r.EXPECT().CountActiveTargets(mock.Anything, testSourceID).Return(int64(1), nil).Once()
			},
			err: datasource.ErrTargetDeactivateNotAllowed,
		},
		"more than one active target": {
			input: struct{ active bool }{active: true},
			mock: func(r *repo.MockRepository) {
				r.EXPECT().DetailDataSource(mock.Anything, testSourceID).Return(testSource(model.SourceStatusActive), nil).Once()
				r.EXPECT().CountActiveTargets(mock.Anything, testSourceID).Return(int64(2), nil).Once()
			},
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			uc, r, _ := newDatasourceUC(t)
			if tc.mock != nil {
				tc.mock(r)
			}

			err := uc.ensureCanRemoveActiveTarget(context.Background(), testSourceID, tc.input.active, datasource.ErrTargetDeactivateNotAllowed)
			require.ErrorIs(t, err, tc.err)
		})
	}
}

func TestEnsureRuntimePrerequisites(t *testing.T) {
	errRepo := errors.New("repo")
	activeTarget := testCrawlTarget(true)

	tcs := map[string]struct {
		input model.DataSource
		mock  func(*repo.MockRepository)
		err   error
	}{
		"crawl missing interval": {
			input: func() model.DataSource {
				source := testSource(model.SourceStatusReady)
				source.CrawlIntervalMinutes = nil
				return source
			}(),
			err: datasource.ErrActivateNotAllowed,
		},
		"crawl list targets error": {
			input: testSource(model.SourceStatusReady),
			mock: func(r *repo.MockRepository) {
				r.EXPECT().ListTargets(mock.Anything, repo.ListTargetsOptions{DataSourceID: testSourceID}).Return(nil, errRepo).Once()
			},
			err: datasource.ErrUpdateFailed,
		},
		"crawl no active targets": {
			input: testSource(model.SourceStatusReady),
			mock: func(r *repo.MockRepository) {
				target := testCrawlTarget(false)
				r.EXPECT().ListTargets(mock.Anything, repo.ListTargetsOptions{DataSourceID: testSourceID}).Return([]model.CrawlTarget{target}, nil).Once()
				r.EXPECT().GetLatestDryrunByTarget(mock.Anything, testTargetID).Return(model.DryrunResult{ID: "dryrun-1", Status: model.DryrunStatusSuccess}, nil).Once()
			},
			err: datasource.ErrActivateNotAllowed,
		},
		"crawl dryrun lookup error": {
			input: testSource(model.SourceStatusReady),
			mock: func(r *repo.MockRepository) {
				r.EXPECT().ListTargets(mock.Anything, repo.ListTargetsOptions{DataSourceID: testSourceID}).Return([]model.CrawlTarget{activeTarget}, nil).Once()
				r.EXPECT().GetLatestDryrunByTarget(mock.Anything, testTargetID).Return(model.DryrunResult{}, errRepo).Once()
			},
			err: datasource.ErrUpdateFailed,
		},
		"crawl dryrun failed": {
			input: testSource(model.SourceStatusReady),
			mock: func(r *repo.MockRepository) {
				r.EXPECT().ListTargets(mock.Anything, repo.ListTargetsOptions{DataSourceID: testSourceID}).Return([]model.CrawlTarget{activeTarget}, nil).Once()
				r.EXPECT().GetLatestDryrunByTarget(mock.Anything, testTargetID).Return(model.DryrunResult{ID: "dryrun-1", Status: model.DryrunStatusFailed}, nil).Once()
			},
			err: datasource.ErrActivateNotAllowed,
		},
		"crawl success": {
			input: testSource(model.SourceStatusReady),
			mock: func(r *repo.MockRepository) {
				r.EXPECT().ListTargets(mock.Anything, repo.ListTargetsOptions{DataSourceID: testSourceID}).Return([]model.CrawlTarget{activeTarget}, nil).Once()
				r.EXPECT().GetLatestDryrunByTarget(mock.Anything, testTargetID).Return(model.DryrunResult{ID: "dryrun-1", Status: model.DryrunStatusSuccess}, nil).Once()
			},
		},
		"webhook missing secret": {
			input: func() model.DataSource {
				source := testSource(model.SourceStatusReady)
				source.SourceCategory = model.SourceCategoryPassive
				source.SourceType = model.SourceTypeWebhook
				source.WebhookSecretEncrypted = ""
				return source
			}(),
			err: datasource.ErrActivateNotAllowed,
		},
		"webhook success": {
			input: func() model.DataSource {
				source := testSource(model.SourceStatusReady)
				source.SourceCategory = model.SourceCategoryPassive
				source.SourceType = model.SourceTypeWebhook
				return source
			}(),
		},
		"passive unsupported": {
			input: func() model.DataSource {
				source := testSource(model.SourceStatusReady)
				source.SourceCategory = model.SourceCategoryPassive
				source.SourceType = model.SourceTypeFileUpload
				return source
			}(),
			err: datasource.ErrActivateNotAllowed,
		},
		"unknown category": {
			input: func() model.DataSource {
				source := testSource(model.SourceStatusReady)
				source.SourceCategory = "bad"
				return source
			}(),
			err: datasource.ErrActivateNotAllowed,
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			uc, r, _ := newDatasourceUC(t)
			if tc.mock != nil {
				tc.mock(r)
			}

			err := uc.ensureRuntimePrerequisites(context.Background(), tc.input, datasource.ErrActivateNotAllowed)
			require.ErrorIs(t, err, tc.err)
		})
	}
}

func TestCancelProjectRuntime(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	tcs := map[string]struct {
		mock func(*execution.MockUseCase)
		nil  bool
		err  error
	}{
		"nil exec": {nil: true},
		"exec returns error": {
			mock: func(execUC *execution.MockUseCase) {
				execUC.EXPECT().CancelProjectRuntime(mock.Anything, execution.CancelProjectRuntimeInput{ProjectID: testProjectID, Reason: "pause", CanceledAt: now}).Return(errors.New("exec")).Once()
			},
			err: errors.New("exec"),
		},
		"success": {
			mock: func(execUC *execution.MockUseCase) {
				execUC.EXPECT().CancelProjectRuntime(mock.Anything, execution.CancelProjectRuntimeInput{ProjectID: testProjectID, Reason: "pause", CanceledAt: now}).Return(nil).Once()
			},
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			uc, _, _ := newDatasourceUC(t)
			if tc.nil {
				uc.exec = nil
			} else {
				execUC := execution.NewMockUseCase(t)
				tc.mock(execUC)
				uc.exec = execUC
			}

			err := uc.cancelProjectRuntime(context.Background(), " "+testProjectID+" ", " pause ", now)
			if tc.err != nil {
				require.EqualError(t, err, tc.err.Error())
				return
			}
			require.NoError(t, err)
		})
	}
}
