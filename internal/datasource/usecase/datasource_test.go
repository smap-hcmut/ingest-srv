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

	"github.com/aarondl/sqlboiler/v4/types"
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

func newDatasourceUC(t *testing.T) (*implUseCase, *repo.MockRepository, *microservice.MockProjectUseCase) {
	t.Helper()
	l := log.NewZapLogger(log.ZapConfig{Level: log.LevelFatal, Mode: log.ModeProduction, Encoding: log.EncodingJSON})
	r := repo.NewMockRepository(t)
	project := microservice.NewMockProjectUseCase(t)
	return &implUseCase{l: l, repo: r, project: project}, r, project
}

func testSource(status model.SourceStatus) model.DataSource {
	mode := model.CrawlModeNormal
	interval := 10
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	return model.DataSource{
		ID:                     testSourceID,
		ProjectID:              testProjectID,
		Name:                   "source",
		SourceType:             model.SourceTypeTikTok,
		SourceCategory:         model.SourceCategoryCrawl,
		Status:                 status,
		OnboardingStatus:       model.OnboardingStatusConfirmed,
		DryrunStatus:           model.DryrunStatusSuccess,
		CrawlMode:              &mode,
		CrawlIntervalMinutes:   &interval,
		WebhookID:              "webhook",
		WebhookSecretEncrypted: "secret",
		CreatedAt:              now,
		UpdatedAt:              now,
	}
}

func testCrawlTarget(active bool) model.CrawlTarget {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	return model.CrawlTarget{
		ID:                   testTargetID,
		DataSourceID:         testSourceID,
		TargetType:           model.TargetTypeKeyword,
		Values:               []string{"vinfast"},
		IsActive:             active,
		CrawlIntervalMinutes: 10,
		CreatedAt:            now,
		UpdatedAt:            now,
	}
}

func TestNew(t *testing.T) {
	tcs := map[string]struct {
		input  struct{}
		mock   struct{}
		output datasource.UseCase
		err    error
	}{
		"success": {},
	}

	for name := range tcs {
		t.Run(name, func(t *testing.T) {
			uc, r, project := newDatasourceUC(t)

			got := New(uc.l, r, project, nil)

			require.NotNil(t, got)
		})
	}
}

func TestCreate(t *testing.T) {
	type mockCreate struct {
		projectCalled bool
		projectOutput microservice.ProjectDetail
		projectErr    error
		repoCalled    bool
		repoInput     repo.CreateDataSourceOptions
		repoOutput    model.DataSource
		repoErr       error
	}

	tcs := map[string]struct {
		input  datasource.CreateInput
		mock   mockCreate
		output datasource.CreateOutput
		err    error
	}{
		"success": {
			input: datasource.CreateInput{ProjectID: testProjectID, Name: " source ", SourceType: string(model.SourceTypeTikTok), CrawlMode: string(model.CrawlModeNormal), CrawlIntervalMinutes: 10},
			mock: mockCreate{
				projectCalled: true,
				projectOutput: microservice.ProjectDetail{ID: testProjectID, Status: microservice.ProjectStatusActive},
				repoCalled:    true,
				repoInput:     repo.CreateDataSourceOptions{ProjectID: testProjectID, Name: "source", SourceType: string(model.SourceTypeTikTok), CrawlMode: string(model.CrawlModeNormal), CrawlIntervalMinutes: 10},
				repoOutput:    testSource(model.SourceStatusPending),
			},
			output: datasource.CreateOutput{DataSource: testSource(model.SourceStatusPending)},
		},
		"invalid_input": {
			input: datasource.CreateInput{Name: "source"},
			err:   datasource.ErrProjectIDRequired,
		},
		"project_archived": {
			input: datasource.CreateInput{ProjectID: testProjectID, Name: "source", SourceType: string(model.SourceTypeWebhook)},
			mock:  mockCreate{projectCalled: true, projectOutput: microservice.ProjectDetail{ID: testProjectID, Status: microservice.ProjectStatusArchived}},
			err:   datasource.ErrProjectArchived,
		},
		"repo_error": {
			input: datasource.CreateInput{ProjectID: testProjectID, Name: "source", SourceType: string(model.SourceTypeWebhook)},
			mock:  mockCreate{projectCalled: true, projectOutput: microservice.ProjectDetail{ID: testProjectID, Status: microservice.ProjectStatusActive}, repoCalled: true, repoInput: repo.CreateDataSourceOptions{ProjectID: testProjectID, Name: "source", SourceType: string(model.SourceTypeWebhook)}, repoErr: errors.New("db")},
			err:   datasource.ErrCreateFailed,
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			uc, r, project := newDatasourceUC(t)
			if tc.mock.projectCalled {
				project.EXPECT().Detail(context.Background(), testProjectID).Return(tc.mock.projectOutput, tc.mock.projectErr)
			}
			if tc.mock.repoCalled {
				r.EXPECT().CreateDataSource(context.Background(), tc.mock.repoInput).Return(tc.mock.repoOutput, tc.mock.repoErr)
			}

			output, err := uc.Create(context.Background(), tc.input)

			require.ErrorIs(t, err, tc.err)
			require.Equal(t, tc.output, output)
		})
	}
}

func TestDetail(t *testing.T) {
	type mockDetail struct {
		isCalled bool
		input    string
		output   model.DataSource
		err      error
	}

	tcs := map[string]struct {
		input  string
		mock   mockDetail
		output datasource.DetailOutput
		err    error
	}{
		"success":     {input: " " + testSourceID + " ", mock: mockDetail{isCalled: true, input: testSourceID, output: testSource(model.SourceStatusReady)}, output: datasource.DetailOutput{DataSource: testSource(model.SourceStatusReady)}},
		"empty":       {err: datasource.ErrNotFound},
		"repo_error":  {input: testSourceID, mock: mockDetail{isCalled: true, input: testSourceID, err: errors.New("db")}, err: datasource.ErrNotFound},
		"empty_model": {input: testSourceID, mock: mockDetail{isCalled: true, input: testSourceID}, err: datasource.ErrNotFound},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			uc, r, _ := newDatasourceUC(t)
			if tc.mock.isCalled {
				r.EXPECT().DetailDataSource(context.Background(), tc.mock.input).Return(tc.mock.output, tc.mock.err)
			}

			output, err := uc.Detail(context.Background(), tc.input)

			require.ErrorIs(t, err, tc.err)
			require.Equal(t, tc.output, output)
		})
	}
}

func TestList(t *testing.T) {
	type mockList struct {
		isCalled bool
		input    repo.GetDataSourcesOptions
		output   []model.DataSource
		pag      paginator.Paginator
		err      error
	}

	tcs := map[string]struct {
		input  datasource.ListInput
		mock   mockList
		output datasource.ListOutput
		err    error
	}{
		"success": {
			input:  datasource.ListInput{ProjectID: " " + testProjectID + " ", SourceType: string(model.SourceTypeTikTok), SourceCategory: string(model.SourceCategoryCrawl), Paginator: paginator.PaginateQuery{Page: 1, Limit: 10}},
			mock:   mockList{isCalled: true, input: repo.GetDataSourcesOptions{ProjectID: testProjectID, SourceType: string(model.SourceTypeTikTok), SourceCategory: string(model.SourceCategoryCrawl), Paginator: paginator.PaginateQuery{Page: 1, Limit: 10}}, output: []model.DataSource{testSource(model.SourceStatusReady)}, pag: paginator.Paginator{Total: 1}},
			output: datasource.ListOutput{DataSources: []model.DataSource{testSource(model.SourceStatusReady)}, Paginator: paginator.Paginator{Total: 1}},
		},
		"invalid_source_type": {input: datasource.ListInput{SourceType: "bad"}, err: datasource.ErrInvalidSourceType},
		"repo_error":          {mock: mockList{isCalled: true, input: repo.GetDataSourcesOptions{Paginator: paginator.PaginateQuery{Page: 1, Limit: 15}}, err: errors.New("db")}, err: datasource.ErrListFailed},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			uc, r, _ := newDatasourceUC(t)
			if tc.mock.isCalled {
				r.EXPECT().GetDataSources(context.Background(), tc.mock.input).Return(tc.mock.output, tc.mock.pag, tc.mock.err)
			}

			output, err := uc.List(context.Background(), tc.input)

			require.ErrorIs(t, err, tc.err)
			require.Equal(t, tc.output, output)
		})
	}
}

func TestUpdate(t *testing.T) {
	type mockUpdate struct {
		detailCalled bool
		detailOutput model.DataSource
		detailErr    error
		targetCalled bool
		targetOutput []model.CrawlTarget
		targetErr    error
		updateCalled bool
		updateInput  repo.UpdateDataSourceOptions
		updateOutput model.DataSource
		updateErr    error
	}

	tcs := map[string]struct {
		input  datasource.UpdateInput
		mock   mockUpdate
		output datasource.UpdateOutput
		err    error
	}{
		"success": {
			input:  datasource.UpdateInput{ID: testSourceID, Name: " next ", Config: []byte(`{"k":"v"}`)},
			mock:   mockUpdate{detailCalled: true, detailOutput: testSource(model.SourceStatusReady), targetCalled: true, updateCalled: true, updateInput: repo.UpdateDataSourceOptions{ID: testSourceID, Name: "next", Config: []byte(`{"k":"v"}`), DryrunStatus: string(model.DryrunStatusNotRequired), DryrunLastResultID: ""}, updateOutput: testSource(model.SourceStatusPending)},
			output: datasource.UpdateOutput{DataSource: testSource(model.SourceStatusPending)},
		},
		"invalid_input": {input: datasource.UpdateInput{}, err: datasource.ErrNotFound},
		"not_found":     {input: datasource.UpdateInput{ID: testSourceID}, mock: mockUpdate{detailCalled: true, detailErr: errors.New("db")}, err: datasource.ErrNotFound},
		"empty_current": {input: datasource.UpdateInput{ID: testSourceID}, mock: mockUpdate{detailCalled: true}, err: datasource.ErrNotFound},
		"dryrun_guard_error": {
			input: datasource.UpdateInput{ID: testSourceID},
			mock:  mockUpdate{detailCalled: true, detailOutput: testSource(model.SourceStatusReady), targetCalled: true, targetErr: errors.New("db")},
			err:   datasource.ErrUpdateFailed,
		},
		"active_runtime_change_not_allowed": {
			input: datasource.UpdateInput{ID: testSourceID, Config: []byte(`{"k":"v"}`)},
			mock:  mockUpdate{detailCalled: true, detailOutput: testSource(model.SourceStatusActive), targetCalled: true},
			err:   datasource.ErrUpdateNotAllowed,
		},
		"repo_error": {
			input: datasource.UpdateInput{ID: testSourceID},
			mock:  mockUpdate{detailCalled: true, detailOutput: testSource(model.SourceStatusReady), targetCalled: true, updateCalled: true, updateInput: repo.UpdateDataSourceOptions{ID: testSourceID}, updateErr: errors.New("db")},
			err:   datasource.ErrUpdateFailed,
		},
		"empty_update_result": {
			input: datasource.UpdateInput{ID: testSourceID},
			mock:  mockUpdate{detailCalled: true, detailOutput: testSource(model.SourceStatusReady), targetCalled: true, updateCalled: true, updateInput: repo.UpdateDataSourceOptions{ID: testSourceID}},
			err:   datasource.ErrNotFound,
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			uc, r, _ := newDatasourceUC(t)
			if tc.mock.detailCalled {
				r.EXPECT().DetailDataSource(context.Background(), testSourceID).Return(tc.mock.detailOutput, tc.mock.detailErr)
			}
			if tc.mock.targetCalled {
				r.EXPECT().ListTargets(context.Background(), repo.ListTargetsOptions{DataSourceID: testSourceID}).Return(tc.mock.targetOutput, tc.mock.targetErr)
			}
			if tc.mock.updateCalled {
				r.EXPECT().UpdateDataSource(context.Background(), tc.mock.updateInput).Return(tc.mock.updateOutput, tc.mock.updateErr)
			}

			output, err := uc.Update(context.Background(), tc.input)

			require.ErrorIs(t, err, tc.err)
			require.Equal(t, tc.output, output)
		})
	}
}

func TestArchive(t *testing.T) {
	type mockArchive struct {
		detailCalled  bool
		detailOutput  model.DataSource
		detailErr     error
		targetCalled  bool
		targetErr     error
		archiveCalled bool
		archiveErr    error
	}

	tcs := map[string]struct {
		input  string
		mock   mockArchive
		output struct{}
		err    error
	}{
		"success":       {input: testSourceID, mock: mockArchive{detailCalled: true, detailOutput: testSource(model.SourceStatusReady), targetCalled: true, archiveCalled: true}},
		"invalid_input": {err: datasource.ErrNotFound},
		"detail_error":  {input: testSourceID, mock: mockArchive{detailCalled: true, detailErr: errors.New("db")}, err: datasource.ErrNotFound},
		"empty_current": {input: testSourceID, mock: mockArchive{detailCalled: true}, err: datasource.ErrNotFound},
		"dryrun_guard_error": {
			input: testSourceID,
			mock:  mockArchive{detailCalled: true, detailOutput: testSource(model.SourceStatusReady), targetCalled: true, targetErr: errors.New("db")},
			err:   datasource.ErrUpdateFailed,
		},
		"already_archived": {input: testSourceID, mock: mockArchive{detailCalled: true, detailOutput: testSource(model.SourceStatusArchived), targetCalled: true}},
		"repo_not_found":   {input: testSourceID, mock: mockArchive{detailCalled: true, detailOutput: testSource(model.SourceStatusReady), targetCalled: true, archiveCalled: true, archiveErr: repo.ErrFailedToGet}, err: datasource.ErrNotFound},
		"repo_error":       {input: testSourceID, mock: mockArchive{detailCalled: true, detailOutput: testSource(model.SourceStatusReady), targetCalled: true, archiveCalled: true, archiveErr: errors.New("db")}, err: datasource.ErrDeleteFailed},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			uc, r, _ := newDatasourceUC(t)
			if tc.mock.detailCalled {
				r.EXPECT().DetailDataSource(context.Background(), tc.input).Return(tc.mock.detailOutput, tc.mock.detailErr)
			}
			if tc.mock.targetCalled {
				r.EXPECT().ListTargets(context.Background(), repo.ListTargetsOptions{DataSourceID: tc.mock.detailOutput.ID}).Return(nil, tc.mock.targetErr)
			}
			if tc.mock.archiveCalled {
				r.EXPECT().ArchiveDataSource(context.Background(), tc.input).Return(tc.mock.archiveErr)
			}

			err := uc.Archive(context.Background(), tc.input)

			require.ErrorIs(t, err, tc.err)
		})
	}
}

func TestDelete(t *testing.T) {
	type mockDelete struct {
		detailCalled bool
		detailOutput model.DataSource
		detailErr    error
		targetCalled bool
		targetErr    error
		deleteCalled bool
		deleteErr    error
	}

	tcs := map[string]struct {
		input  string
		mock   mockDelete
		output struct{}
		err    error
	}{
		"success":       {input: testSourceID, mock: mockDelete{detailCalled: true, detailOutput: testSource(model.SourceStatusArchived), targetCalled: true, deleteCalled: true}},
		"invalid_input": {err: datasource.ErrNotFound},
		"detail_error":  {input: testSourceID, mock: mockDelete{detailCalled: true, detailErr: errors.New("db")}, err: datasource.ErrNotFound},
		"empty_current": {input: testSourceID, mock: mockDelete{detailCalled: true}, err: datasource.ErrNotFound},
		"dryrun_guard_error": {input: testSourceID, mock: mockDelete{
			detailCalled: true,
			detailOutput: testSource(model.SourceStatusArchived),
			targetCalled: true,
			targetErr:    errors.New("db"),
		}, err: datasource.ErrUpdateFailed},
		"requires_archived": {input: testSourceID, mock: mockDelete{detailCalled: true, detailOutput: testSource(model.SourceStatusReady), targetCalled: true}, err: datasource.ErrDeleteRequiresArchived},
		"delete_not_found":  {input: testSourceID, mock: mockDelete{detailCalled: true, detailOutput: testSource(model.SourceStatusArchived), targetCalled: true, deleteCalled: true, deleteErr: repo.ErrFailedToGet}, err: datasource.ErrNotFound},
		"delete_repo_error": {input: testSourceID, mock: mockDelete{detailCalled: true, detailOutput: testSource(model.SourceStatusArchived), targetCalled: true, deleteCalled: true, deleteErr: errors.New("db")}, err: datasource.ErrDeleteFailed},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			uc, r, _ := newDatasourceUC(t)
			if tc.mock.detailCalled {
				r.EXPECT().DetailDataSource(context.Background(), tc.input).Return(tc.mock.detailOutput, tc.mock.detailErr)
			}
			if tc.mock.targetCalled {
				r.EXPECT().ListTargets(context.Background(), repo.ListTargetsOptions{DataSourceID: tc.mock.detailOutput.ID}).Return(nil, tc.mock.targetErr)
			}
			if tc.mock.deleteCalled {
				r.EXPECT().DeleteDataSource(context.Background(), tc.input).Return(tc.mock.deleteErr)
			}

			err := uc.Delete(context.Background(), tc.input)

			require.ErrorIs(t, err, tc.err)
		})
	}
}

func TestMarkDryrunRunning(t *testing.T) {
	type mockData struct {
		detail model.DataSource
		err    error
		update repo.UpdateDataSourceOptions
		output model.DataSource
		upErr  error
	}

	tcs := map[string]struct {
		input  datasource.MarkDryrunRunningInput
		mock   mockData
		output datasource.MarkDryrunRunningOutput
		err    error
	}{
		"success":       {input: datasource.MarkDryrunRunningInput{ID: testSourceID, DryrunLastResultID: "result-1"}, mock: mockData{detail: testSource(model.SourceStatusReady), update: repo.UpdateDataSourceOptions{ID: testSourceID, DryrunStatus: string(model.DryrunStatusRunning), DryrunLastResultID: "result-1", Status: string(model.SourceStatusPending)}, output: testSource(model.SourceStatusPending)}, output: datasource.MarkDryrunRunningOutput{DataSource: testSource(model.SourceStatusPending)}},
		"active_source": {input: datasource.MarkDryrunRunningInput{ID: testSourceID, DryrunLastResultID: "result-1"}, mock: mockData{detail: testSource(model.SourceStatusActive), update: repo.UpdateDataSourceOptions{ID: testSourceID, DryrunStatus: string(model.DryrunStatusRunning), DryrunLastResultID: "result-1"}, output: testSource(model.SourceStatusActive)}, output: datasource.MarkDryrunRunningOutput{DataSource: testSource(model.SourceStatusActive)}},
		"detail_error":  {input: datasource.MarkDryrunRunningInput{ID: testSourceID}, mock: mockData{err: errors.New("db")}, err: datasource.ErrUpdateFailed},
		"empty_model":   {input: datasource.MarkDryrunRunningInput{ID: testSourceID}, err: datasource.ErrNotFound},
		"update_error":  {input: datasource.MarkDryrunRunningInput{ID: testSourceID}, mock: mockData{detail: testSource(model.SourceStatusReady), update: repo.UpdateDataSourceOptions{ID: testSourceID, DryrunStatus: string(model.DryrunStatusRunning), Status: string(model.SourceStatusPending)}, upErr: errors.New("db")}, err: datasource.ErrUpdateFailed},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			uc, r, _ := newDatasourceUC(t)
			r.EXPECT().DetailDataSource(context.Background(), tc.input.ID).Return(tc.mock.detail, tc.mock.err)
			if tc.mock.detail.ID != "" {
				r.EXPECT().UpdateDataSource(context.Background(), tc.mock.update).Return(tc.mock.output, tc.mock.upErr)
			}

			output, err := uc.MarkDryrunRunning(context.Background(), tc.input)

			require.ErrorIs(t, err, tc.err)
			require.Equal(t, tc.output, output)
		})
	}
}

func TestApplyDryrunResult(t *testing.T) {
	type mockData struct {
		detail model.DataSource
		err    error
		update repo.UpdateDataSourceOptions
		output model.DataSource
		upErr  error
	}

	tcs := map[string]struct {
		input  datasource.ApplyDryrunResultInput
		mock   mockData
		output datasource.ApplyDryrunResultOutput
		err    error
	}{
		"success_failed":  {input: datasource.ApplyDryrunResultInput{ID: testSourceID, DryrunLastResultID: "result-1", DryrunStatus: string(model.DryrunStatusFailed)}, mock: mockData{detail: testSource(model.SourceStatusReady), update: repo.UpdateDataSourceOptions{ID: testSourceID, DryrunStatus: string(model.DryrunStatusFailed), DryrunLastResultID: "result-1", Status: string(model.SourceStatusPending)}, output: testSource(model.SourceStatusPending)}, output: datasource.ApplyDryrunResultOutput{DataSource: testSource(model.SourceStatusPending)}},
		"success_success": {input: datasource.ApplyDryrunResultInput{ID: testSourceID, DryrunLastResultID: "result-1", DryrunStatus: string(model.DryrunStatusSuccess)}, mock: mockData{detail: testSource(model.SourceStatusPending), update: repo.UpdateDataSourceOptions{ID: testSourceID, DryrunStatus: string(model.DryrunStatusSuccess), DryrunLastResultID: "result-1", Status: string(model.SourceStatusReady)}, output: testSource(model.SourceStatusReady)}, output: datasource.ApplyDryrunResultOutput{DataSource: testSource(model.SourceStatusReady)}},
		"active_source":   {input: datasource.ApplyDryrunResultInput{ID: testSourceID, DryrunLastResultID: "result-1", DryrunStatus: string(model.DryrunStatusSuccess)}, mock: mockData{detail: testSource(model.SourceStatusActive), update: repo.UpdateDataSourceOptions{ID: testSourceID, DryrunStatus: string(model.DryrunStatusSuccess), DryrunLastResultID: "result-1"}, output: testSource(model.SourceStatusActive)}, output: datasource.ApplyDryrunResultOutput{DataSource: testSource(model.SourceStatusActive)}},
		"detail_error":    {input: datasource.ApplyDryrunResultInput{ID: testSourceID}, mock: mockData{err: errors.New("db")}, err: datasource.ErrUpdateFailed},
		"empty_model":     {input: datasource.ApplyDryrunResultInput{ID: testSourceID}, err: datasource.ErrNotFound},
		"update_error":    {input: datasource.ApplyDryrunResultInput{ID: testSourceID, DryrunStatus: string(model.DryrunStatusSuccess)}, mock: mockData{detail: testSource(model.SourceStatusPending), update: repo.UpdateDataSourceOptions{ID: testSourceID, DryrunStatus: string(model.DryrunStatusSuccess), Status: string(model.SourceStatusReady)}, upErr: errors.New("db")}, err: datasource.ErrUpdateFailed},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			uc, r, _ := newDatasourceUC(t)
			r.EXPECT().DetailDataSource(context.Background(), tc.input.ID).Return(tc.mock.detail, tc.mock.err)
			if tc.mock.detail.ID != "" {
				r.EXPECT().UpdateDataSource(context.Background(), tc.mock.update).Return(tc.mock.output, tc.mock.upErr)
			}

			output, err := uc.ApplyDryrunResult(context.Background(), tc.input)

			require.ErrorIs(t, err, tc.err)
			require.Equal(t, tc.output, output)
		})
	}
}

func TestCreateKeywordTarget(t *testing.T) {
	testCreateTarget(t, func(uc *implUseCase, ctx context.Context, input datasource.CreateTargetGroupInput) (datasource.CreateTargetOutput, error) {
		return uc.CreateKeywordTarget(ctx, input)
	}, model.TargetTypeKeyword, []string{" vinfast ", "vinfast"}, model.TypesJSONFromStringSlice([]string{"vinfast"}))
}

func TestCreateProfileTarget(t *testing.T) {
	testCreateTarget(t, func(uc *implUseCase, ctx context.Context, input datasource.CreateTargetGroupInput) (datasource.CreateTargetOutput, error) {
		return uc.CreateProfileTarget(ctx, input)
	}, model.TargetTypeProfile, []string{"https://example.com/u"}, model.TypesJSONFromStringSlice([]string{"https://example.com/u"}))
}

func TestCreatePostTarget(t *testing.T) {
	testCreateTarget(t, func(uc *implUseCase, ctx context.Context, input datasource.CreateTargetGroupInput) (datasource.CreateTargetOutput, error) {
		return uc.CreatePostTarget(ctx, input)
	}, model.TargetTypePostURL, []string{"https://example.com/p"}, model.TypesJSONFromStringSlice([]string{"https://example.com/p"}))
}

func testCreateTarget(t *testing.T, call func(*implUseCase, context.Context, datasource.CreateTargetGroupInput) (datasource.CreateTargetOutput, error), targetType model.TargetType, values []string, repoValues types.JSON) {
	t.Helper()
	type mockCreateTarget struct {
		detailCalled bool
		detailOutput model.DataSource
		detailErr    error
		createCalled bool
		createInput  repo.CreateTargetOptions
		createOutput model.CrawlTarget
		createErr    error
	}

	tcs := map[string]struct {
		input  datasource.CreateTargetGroupInput
		mock   mockCreateTarget
		output datasource.CreateTargetOutput
		err    error
	}{
		"success": {
			input:  datasource.CreateTargetGroupInput{DataSourceID: testSourceID, Values: values, CrawlIntervalMinutes: 10},
			mock:   mockCreateTarget{detailCalled: true, detailOutput: testSource(model.SourceStatusReady), createCalled: true, createInput: repo.CreateTargetOptions{DataSourceID: testSourceID, TargetType: string(targetType), Values: repoValues, IsActive: false, CrawlIntervalMinutes: 10}, createOutput: testCrawlTarget(false)},
			output: datasource.CreateTargetOutput{Target: testCrawlTarget(false)},
		},
		"invalid_input": {input: datasource.CreateTargetGroupInput{}, err: datasource.ErrProjectIDRequired},
		"detail_error": {
			input: datasource.CreateTargetGroupInput{DataSourceID: testSourceID, Values: values, CrawlIntervalMinutes: 10},
			mock:  mockCreateTarget{detailCalled: true, detailErr: errors.New("db")},
			err:   datasource.ErrTargetCreateFailed,
		},
		"empty_source": {
			input: datasource.CreateTargetGroupInput{DataSourceID: testSourceID, Values: values, CrawlIntervalMinutes: 10},
			mock:  mockCreateTarget{detailCalled: true},
			err:   datasource.ErrNotFound,
		},
		"archived_source": {
			input: datasource.CreateTargetGroupInput{DataSourceID: testSourceID, Values: values, CrawlIntervalMinutes: 10},
			mock:  mockCreateTarget{detailCalled: true, detailOutput: testSource(model.SourceStatusArchived)},
			err:   datasource.ErrSourceArchived,
		},
		"source_not_crawl": {
			input: datasource.CreateTargetGroupInput{DataSourceID: testSourceID, Values: values, CrawlIntervalMinutes: 10},
			mock: mockCreateTarget{detailCalled: true, detailOutput: func() model.DataSource {
				ds := testSource(model.SourceStatusReady)
				ds.SourceCategory = model.SourceCategoryPassive
				return ds
			}()},
			err: datasource.ErrSourceNotCrawl,
		},
		"repo_error": {
			input: datasource.CreateTargetGroupInput{DataSourceID: testSourceID, Values: values, CrawlIntervalMinutes: 10},
			mock:  mockCreateTarget{detailCalled: true, detailOutput: testSource(model.SourceStatusReady), createCalled: true, createInput: repo.CreateTargetOptions{DataSourceID: testSourceID, TargetType: string(targetType), Values: repoValues, IsActive: false, CrawlIntervalMinutes: 10}, createErr: errors.New("db")},
			err:   datasource.ErrTargetCreateFailed,
		},
		"invalid_values": {
			input: datasource.CreateTargetGroupInput{DataSourceID: testSourceID, Values: []string{" "}, CrawlIntervalMinutes: 10},
			mock:  mockCreateTarget{detailCalled: true, detailOutput: testSource(model.SourceStatusReady)},
			err:   datasource.ErrTargetValuesRequired,
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			uc, r, _ := newDatasourceUC(t)
			if tc.mock.detailCalled {
				r.EXPECT().DetailDataSource(context.Background(), testSourceID).Return(tc.mock.detailOutput, tc.mock.detailErr)
			}
			if tc.mock.createCalled {
				r.EXPECT().CreateTarget(context.Background(), tc.mock.createInput).Return(tc.mock.createOutput, tc.mock.createErr)
			}

			output, err := call(uc, context.Background(), tc.input)

			require.ErrorIs(t, err, tc.err)
			require.Equal(t, tc.output, output)
		})
	}
}

func TestDetailTarget(t *testing.T) {
	type mockDetail struct {
		isCalled bool
		input    repo.GetTargetOptions
		output   model.CrawlTarget
		err      error
	}

	tcs := map[string]struct {
		input  datasource.DetailTargetInput
		mock   mockDetail
		output datasource.DetailTargetOutput
		err    error
	}{
		"success":    {input: datasource.DetailTargetInput{DataSourceID: testSourceID, ID: testTargetID}, mock: mockDetail{isCalled: true, input: repo.GetTargetOptions{DataSourceID: testSourceID, ID: testTargetID}, output: testCrawlTarget(true)}, output: datasource.DetailTargetOutput{Target: testCrawlTarget(true)}},
		"invalid":    {err: datasource.ErrTargetNotFound},
		"repo_error": {input: datasource.DetailTargetInput{DataSourceID: testSourceID, ID: testTargetID}, mock: mockDetail{isCalled: true, input: repo.GetTargetOptions{DataSourceID: testSourceID, ID: testTargetID}, err: errors.New("db")}, err: datasource.ErrTargetNotFound},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			uc, r, _ := newDatasourceUC(t)
			if tc.mock.isCalled {
				r.EXPECT().GetTarget(context.Background(), tc.mock.input).Return(tc.mock.output, tc.mock.err)
			}

			output, err := uc.DetailTarget(context.Background(), tc.input)

			require.ErrorIs(t, err, tc.err)
			require.Equal(t, tc.output, output)
		})
	}
}

func TestListTargets(t *testing.T) {
	active := true
	type mockList struct {
		isCalled bool
		input    repo.ListTargetsOptions
		output   []model.CrawlTarget
		err      error
	}

	tcs := map[string]struct {
		input  datasource.ListTargetsInput
		mock   mockList
		output datasource.ListTargetsOutput
		err    error
	}{
		"success":    {input: datasource.ListTargetsInput{DataSourceID: testSourceID, TargetType: string(model.TargetTypeKeyword), IsActive: &active}, mock: mockList{isCalled: true, input: repo.ListTargetsOptions{DataSourceID: testSourceID, TargetType: string(model.TargetTypeKeyword), IsActive: &active}, output: []model.CrawlTarget{testCrawlTarget(true)}}, output: datasource.ListTargetsOutput{Targets: []model.CrawlTarget{testCrawlTarget(true)}}},
		"invalid":    {input: datasource.ListTargetsInput{DataSourceID: testSourceID, TargetType: "bad"}, err: datasource.ErrInvalidTargetType},
		"repo_error": {input: datasource.ListTargetsInput{DataSourceID: testSourceID}, mock: mockList{isCalled: true, input: repo.ListTargetsOptions{DataSourceID: testSourceID}, err: errors.New("db")}, err: datasource.ErrTargetListFailed},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			uc, r, _ := newDatasourceUC(t)
			if tc.mock.isCalled {
				r.EXPECT().ListTargets(context.Background(), tc.mock.input).Return(tc.mock.output, tc.mock.err)
			}

			output, err := uc.ListTargets(context.Background(), tc.input)

			require.ErrorIs(t, err, tc.err)
			require.Equal(t, tc.output, output)
		})
	}
}

func TestActivateDataSource(t *testing.T) {
	type mockData struct {
		detail    model.DataSource
		detailErr error
		targets   []model.CrawlTarget
		targetErr error
		latest    model.DryrunResult
		latestErr error
		update    model.DataSource
		updateErr error
	}

	tcs := map[string]struct {
		input  string
		mock   mockData
		output datasource.ActivateOutput
		err    error
	}{
		"success": {
			input:  testSourceID,
			mock:   mockData{detail: testSource(model.SourceStatusReady), targets: []model.CrawlTarget{testCrawlTarget(true)}, latest: model.DryrunResult{ID: "dryrun-1", Status: model.DryrunStatusSuccess}, update: testSource(model.SourceStatusActive)},
			output: datasource.ActivateOutput{DataSource: testSource(model.SourceStatusActive)},
		},
		"invalid":           {err: datasource.ErrNotFound},
		"detail_error":      {input: testSourceID, mock: mockData{detailErr: errors.New("db")}, err: datasource.ErrNotFound},
		"empty_model":       {input: testSourceID, err: datasource.ErrNotFound},
		"wrong_status":      {input: testSourceID, mock: mockData{detail: testSource(model.SourceStatusPending)}, err: datasource.ErrActivateNotAllowed},
		"runtime_not_ready": {input: testSourceID, mock: mockData{detail: testSource(model.SourceStatusReady), targets: []model.CrawlTarget{}}, err: datasource.ErrActivateNotAllowed},
		"runtime_repo_error": {
			input: testSourceID,
			mock:  mockData{detail: testSource(model.SourceStatusReady), targetErr: errors.New("db")},
			err:   datasource.ErrUpdateFailed,
		},
		"latest_error": {
			input: testSourceID,
			mock:  mockData{detail: testSource(model.SourceStatusReady), targets: []model.CrawlTarget{testCrawlTarget(true)}, latestErr: errors.New("db")},
			err:   datasource.ErrUpdateFailed,
		},
		"update_error": {
			input: testSourceID,
			mock:  mockData{detail: testSource(model.SourceStatusReady), targets: []model.CrawlTarget{testCrawlTarget(true)}, latest: model.DryrunResult{ID: "dryrun-1", Status: model.DryrunStatusSuccess}, updateErr: errors.New("db")},
			err:   datasource.ErrUpdateFailed,
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			uc, r, _ := newDatasourceUC(t)
			if tc.input != "" {
				r.EXPECT().DetailDataSource(context.Background(), testSourceID).Return(tc.mock.detail, tc.mock.detailErr)
			}
			if tc.mock.detail.ID != "" && tc.mock.detail.Status == model.SourceStatusReady {
				r.EXPECT().ListTargets(context.Background(), repo.ListTargetsOptions{DataSourceID: testSourceID}).Return(tc.mock.targets, tc.mock.targetErr)
				if len(tc.mock.targets) > 0 && tc.mock.targetErr == nil {
					r.EXPECT().GetLatestDryrunByTarget(context.Background(), testTargetID).Return(tc.mock.latest, tc.mock.latestErr)
				}
			}
			if tc.err == nil || tc.mock.updateErr != nil {
				r.EXPECT().UpdateDataSource(context.Background(), repo.UpdateDataSourceOptions{ID: testSourceID, Status: string(model.SourceStatusActive), ClearPausedAt: true}).Return(tc.mock.update, tc.mock.updateErr)
			}

			output, err := uc.ActivateDataSource(context.Background(), tc.input)

			require.ErrorIs(t, err, tc.err)
			require.Equal(t, tc.output, output)
		})
	}
}

func TestPauseDataSource(t *testing.T) {
	type mockData struct {
		detail    model.DataSource
		detailErr error
		update    model.DataSource
		updateErr error
	}
	tcs := map[string]struct {
		input  string
		mock   mockData
		output datasource.PauseOutput
		err    error
	}{
		"success":      {input: testSourceID, mock: mockData{detail: testSource(model.SourceStatusActive), update: testSource(model.SourceStatusPaused)}, output: datasource.PauseOutput{DataSource: testSource(model.SourceStatusPaused)}},
		"invalid":      {err: datasource.ErrNotFound},
		"detail_error": {input: testSourceID, mock: mockData{detailErr: errors.New("db")}, err: datasource.ErrNotFound},
		"empty_model":  {input: testSourceID, err: datasource.ErrNotFound},
		"wrong_status": {input: testSourceID, mock: mockData{detail: testSource(model.SourceStatusReady)}, err: datasource.ErrPauseNotAllowed},
		"update_error": {input: testSourceID, mock: mockData{detail: testSource(model.SourceStatusActive), updateErr: errors.New("db")}, err: datasource.ErrUpdateFailed},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			uc, r, _ := newDatasourceUC(t)
			if tc.input != "" {
				r.EXPECT().DetailDataSource(context.Background(), testSourceID).Return(tc.mock.detail, tc.mock.detailErr)
			}
			if tc.err == nil || tc.mock.updateErr != nil {
				r.EXPECT().UpdateDataSource(context.Background(), repo.UpdateDataSourceOptions{ID: testSourceID, Status: string(model.SourceStatusPaused)}).Return(tc.mock.update, tc.mock.updateErr)
			}

			output, err := uc.PauseDataSource(context.Background(), tc.input)

			require.ErrorIs(t, err, tc.err)
			require.Equal(t, tc.output, output)
		})
	}
}

func TestResumeDataSource(t *testing.T) {
	type mockData struct {
		detail    model.DataSource
		detailErr error
		targets   []model.CrawlTarget
		targetErr error
		latest    model.DryrunResult
		latestErr error
		update    model.DataSource
		updateErr error
	}
	tcs := map[string]struct {
		input  string
		mock   mockData
		output datasource.ResumeOutput
		err    error
	}{
		"success":      {input: testSourceID, mock: mockData{detail: testSource(model.SourceStatusPaused), targets: []model.CrawlTarget{testCrawlTarget(true)}, latest: model.DryrunResult{ID: "dryrun-1", Status: model.DryrunStatusSuccess}, update: testSource(model.SourceStatusActive)}, output: datasource.ResumeOutput{DataSource: testSource(model.SourceStatusActive)}},
		"invalid":      {err: datasource.ErrNotFound},
		"detail_error": {input: testSourceID, mock: mockData{detailErr: errors.New("db")}, err: datasource.ErrNotFound},
		"empty_model":  {input: testSourceID, err: datasource.ErrNotFound},
		"wrong_status": {input: testSourceID, mock: mockData{detail: testSource(model.SourceStatusReady)}, err: datasource.ErrResumeNotAllowed},
		"runtime_error": {
			input: testSourceID,
			mock:  mockData{detail: testSource(model.SourceStatusPaused), targetErr: errors.New("db")},
			err:   datasource.ErrUpdateFailed,
		},
		"update_error": {
			input: testSourceID,
			mock:  mockData{detail: testSource(model.SourceStatusPaused), targets: []model.CrawlTarget{testCrawlTarget(true)}, latest: model.DryrunResult{ID: "dryrun-1", Status: model.DryrunStatusSuccess}, updateErr: errors.New("db")},
			err:   datasource.ErrUpdateFailed,
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			uc, r, _ := newDatasourceUC(t)
			if tc.input != "" {
				r.EXPECT().DetailDataSource(context.Background(), testSourceID).Return(tc.mock.detail, tc.mock.detailErr)
			}
			if tc.mock.detail.ID != "" && tc.mock.detail.Status == model.SourceStatusPaused {
				r.EXPECT().ListTargets(context.Background(), repo.ListTargetsOptions{DataSourceID: testSourceID}).Return(tc.mock.targets, tc.mock.targetErr)
				if len(tc.mock.targets) > 0 && tc.mock.targetErr == nil {
					r.EXPECT().GetLatestDryrunByTarget(context.Background(), testTargetID).Return(tc.mock.latest, tc.mock.latestErr)
				}
			}
			if tc.err == nil || tc.mock.updateErr != nil {
				r.EXPECT().UpdateDataSource(context.Background(), repo.UpdateDataSourceOptions{ID: testSourceID, Status: string(model.SourceStatusActive), ClearPausedAt: true}).Return(tc.mock.update, tc.mock.updateErr)
			}

			output, err := uc.ResumeDataSource(context.Background(), tc.input)

			require.ErrorIs(t, err, tc.err)
			require.Equal(t, tc.output, output)
		})
	}
}

func TestUpdateCrawlMode(t *testing.T) {
	type mockData struct {
		detail    model.DataSource
		detailErr error
		update    model.DataSource
		updateErr error
		changeErr error
	}

	tcs := map[string]struct {
		input  datasource.UpdateCrawlModeInput
		mock   mockData
		output datasource.UpdateCrawlModeOutput
		err    error
	}{
		"success": {
			input:  datasource.UpdateCrawlModeInput{ID: testSourceID, CrawlMode: string(model.CrawlModeCrisis), TriggerType: string(model.TriggerTypeManual), Reason: " reason ", EventRef: " event "},
			mock:   mockData{detail: testSource(model.SourceStatusActive), update: testSource(model.SourceStatusActive)},
			output: datasource.UpdateCrawlModeOutput{DataSource: testSource(model.SourceStatusActive)},
		},
		"invalid":         {input: datasource.UpdateCrawlModeInput{ID: testSourceID, CrawlMode: "bad", TriggerType: string(model.TriggerTypeManual)}, err: datasource.ErrInvalidCrawlMode},
		"invalid_trigger": {input: datasource.UpdateCrawlModeInput{ID: testSourceID, CrawlMode: string(model.CrawlModeNormal), TriggerType: "bad"}, err: datasource.ErrCrawlModeNotAllowed},
		"detail_error":    {input: datasource.UpdateCrawlModeInput{ID: testSourceID, CrawlMode: string(model.CrawlModeNormal), TriggerType: string(model.TriggerTypeManual)}, mock: mockData{detailErr: errors.New("db")}, err: datasource.ErrNotFound},
		"empty_model":     {input: datasource.UpdateCrawlModeInput{ID: testSourceID, CrawlMode: string(model.CrawlModeNormal), TriggerType: string(model.TriggerTypeManual)}, err: datasource.ErrNotFound},
		"not_crawl": {input: datasource.UpdateCrawlModeInput{ID: testSourceID, CrawlMode: string(model.CrawlModeNormal), TriggerType: string(model.TriggerTypeManual)}, mock: mockData{detail: func() model.DataSource {
			ds := testSource(model.SourceStatusActive)
			ds.SourceCategory = model.SourceCategoryPassive
			return ds
		}()}, err: datasource.ErrCrawlModeNotAllowed},
		"wrong_status": {
			input: datasource.UpdateCrawlModeInput{ID: testSourceID, CrawlMode: string(model.CrawlModeNormal), TriggerType: string(model.TriggerTypeManual)},
			mock:  mockData{detail: testSource(model.SourceStatusPending)},
			err:   datasource.ErrCrawlModeNotAllowed,
		},
		"invalid_config": {
			input: datasource.UpdateCrawlModeInput{ID: testSourceID, CrawlMode: string(model.CrawlModeNormal), TriggerType: string(model.TriggerTypeManual)},
			mock: mockData{detail: func() model.DataSource {
				ds := testSource(model.SourceStatusActive)
				ds.CrawlIntervalMinutes = nil
				return ds
			}()},
			err: datasource.ErrCrawlModeNotAllowed,
		},
		"update_error": {
			input: datasource.UpdateCrawlModeInput{ID: testSourceID, CrawlMode: string(model.CrawlModeCrisis), TriggerType: string(model.TriggerTypeManual)},
			mock:  mockData{detail: testSource(model.SourceStatusActive), updateErr: errors.New("db")},
			err:   datasource.ErrUpdateFailed,
		},
		"change_error": {
			input: datasource.UpdateCrawlModeInput{ID: testSourceID, CrawlMode: string(model.CrawlModeCrisis), TriggerType: string(model.TriggerTypeManual)},
			mock:  mockData{detail: testSource(model.SourceStatusActive), update: testSource(model.SourceStatusActive), changeErr: errors.New("db")},
			err:   datasource.ErrUpdateFailed,
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			uc, r, _ := newDatasourceUC(t)
			if tc.err != datasource.ErrInvalidCrawlMode && tc.err != datasource.ErrCrawlModeNotAllowed || tc.mock.detail.ID != "" || tc.mock.detailErr != nil {
				r.EXPECT().DetailDataSource(context.Background(), testSourceID).Return(tc.mock.detail, tc.mock.detailErr)
			}
			if tc.err == nil || tc.mock.updateErr != nil || tc.mock.changeErr != nil {
				r.EXPECT().UpdateDataSource(context.Background(), repo.UpdateDataSourceOptions{ID: testSourceID, CrawlMode: string(model.CrawlModeCrisis)}).Return(tc.mock.update, tc.mock.updateErr)
			}
			if tc.err == nil || tc.mock.changeErr != nil {
				r.EXPECT().CreateCrawlModeChange(context.Background(), mock.MatchedBy(func(opt repo.CreateCrawlModeChangeOptions) bool {
					return opt.SourceID == testSourceID &&
						opt.ProjectID == testProjectID &&
						opt.TriggerType == string(model.TriggerTypeManual) &&
						opt.FromMode == string(model.CrawlModeNormal) &&
						opt.ToMode == string(model.CrawlModeCrisis) &&
						opt.FromIntervalMinutes == 10 &&
						opt.ToIntervalMinutes == 10
				})).Return(model.CrawlModeChange{}, tc.mock.changeErr)
			}

			output, err := uc.UpdateCrawlMode(context.Background(), tc.input)

			require.ErrorIs(t, err, tc.err)
			require.Equal(t, tc.output, output)
		})
	}
}

func TestUpdateTarget(t *testing.T) {
	interval := 20
	type mockData struct {
		getCalled     bool
		current       model.CrawlTarget
		getErr        error
		latestCalled  bool
		latest        model.DryrunResult
		latestErr     error
		updateCalled  bool
		updateInput   repo.UpdateTargetOptions
		update        model.CrawlTarget
		updateErr     error
		markCalled    bool
		markDetail    model.DataSource
		markDetailErr error
		markUpdateErr error
	}

	tcs := map[string]struct {
		input  datasource.UpdateTargetInput
		mock   mockData
		output datasource.UpdateTargetOutput
		err    error
	}{
		"success": {
			input:  datasource.UpdateTargetInput{DataSourceID: testSourceID, ID: testTargetID, Values: []string{"new"}, CrawlIntervalMinutes: &interval},
			mock:   mockData{getCalled: true, current: testCrawlTarget(true), latestCalled: true, updateCalled: true, updateInput: repo.UpdateTargetOptions{DataSourceID: testSourceID, ID: testTargetID, Values: model.TypesJSONFromStringSlice([]string{"new"}), IsActive: boolPtr(false), CrawlIntervalMinutes: &interval}, update: testCrawlTarget(false), markCalled: true, markDetail: testSource(model.SourceStatusReady)},
			output: datasource.UpdateTargetOutput{Target: testCrawlTarget(false)},
		},
		"invalid": {err: datasource.ErrTargetNotFound},
		"target_not_found": {
			input: datasource.UpdateTargetInput{DataSourceID: testSourceID, ID: testTargetID},
			mock:  mockData{getCalled: true, getErr: repo.ErrTargetNotFound},
			err:   datasource.ErrTargetNotFound,
		},
		"repo_error": {
			input: datasource.UpdateTargetInput{DataSourceID: testSourceID, ID: testTargetID},
			mock:  mockData{getCalled: true, getErr: errors.New("db")},
			err:   datasource.ErrTargetUpdateFailed,
		},
		"dryrun_running": {
			input: datasource.UpdateTargetInput{DataSourceID: testSourceID, ID: testTargetID},
			mock:  mockData{getCalled: true, current: testCrawlTarget(true), latestCalled: true, latest: model.DryrunResult{ID: "dryrun-1", Status: model.DryrunStatusRunning}},
			err:   datasource.ErrTargetDryrunRunning,
		},
		"invalid_values": {
			input: datasource.UpdateTargetInput{DataSourceID: testSourceID, ID: testTargetID, Values: []string{" "}},
			mock:  mockData{getCalled: true, current: testCrawlTarget(true), latestCalled: true},
			err:   datasource.ErrTargetValuesRequired,
		},
		"update_not_found": {
			input: datasource.UpdateTargetInput{DataSourceID: testSourceID, ID: testTargetID, Values: []string{"new"}},
			mock:  mockData{getCalled: true, current: testCrawlTarget(true), latestCalled: true, updateCalled: true, updateInput: repo.UpdateTargetOptions{DataSourceID: testSourceID, ID: testTargetID, Values: model.TypesJSONFromStringSlice([]string{"new"}), IsActive: boolPtr(false)}, updateErr: repo.ErrTargetNotFound},
			err:   datasource.ErrTargetNotFound,
		},
		"update_error": {
			input: datasource.UpdateTargetInput{DataSourceID: testSourceID, ID: testTargetID, Values: []string{"new"}},
			mock:  mockData{getCalled: true, current: testCrawlTarget(true), latestCalled: true, updateCalled: true, updateInput: repo.UpdateTargetOptions{DataSourceID: testSourceID, ID: testTargetID, Values: model.TypesJSONFromStringSlice([]string{"new"}), IsActive: boolPtr(false)}, updateErr: errors.New("db")},
			err:   datasource.ErrTargetUpdateFailed,
		},
		"mark_detail_error": {
			input: datasource.UpdateTargetInput{DataSourceID: testSourceID, ID: testTargetID, Values: []string{"new"}},
			mock:  mockData{getCalled: true, current: testCrawlTarget(true), latestCalled: true, updateCalled: true, updateInput: repo.UpdateTargetOptions{DataSourceID: testSourceID, ID: testTargetID, Values: model.TypesJSONFromStringSlice([]string{"new"}), IsActive: boolPtr(false)}, update: testCrawlTarget(false), markCalled: true, markDetailErr: errors.New("db")},
			err:   datasource.ErrTargetUpdateFailed,
		},
		"mark_empty_source": {
			input: datasource.UpdateTargetInput{DataSourceID: testSourceID, ID: testTargetID, Values: []string{"new"}},
			mock:  mockData{getCalled: true, current: testCrawlTarget(true), latestCalled: true, updateCalled: true, updateInput: repo.UpdateTargetOptions{DataSourceID: testSourceID, ID: testTargetID, Values: model.TypesJSONFromStringSlice([]string{"new"}), IsActive: boolPtr(false)}, update: testCrawlTarget(false), markCalled: true},
			err:   datasource.ErrNotFound,
		},
		"mark_update_error": {
			input: datasource.UpdateTargetInput{DataSourceID: testSourceID, ID: testTargetID, Values: []string{"new"}},
			mock:  mockData{getCalled: true, current: testCrawlTarget(true), latestCalled: true, updateCalled: true, updateInput: repo.UpdateTargetOptions{DataSourceID: testSourceID, ID: testTargetID, Values: model.TypesJSONFromStringSlice([]string{"new"}), IsActive: boolPtr(false)}, update: testCrawlTarget(false), markCalled: true, markDetail: testSource(model.SourceStatusReady), markUpdateErr: errors.New("db")},
			err:   datasource.ErrTargetUpdateFailed,
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			uc, r, _ := newDatasourceUC(t)
			if tc.mock.getCalled {
				r.EXPECT().GetTarget(context.Background(), repo.GetTargetOptions{DataSourceID: testSourceID, ID: testTargetID}).Return(tc.mock.current, tc.mock.getErr)
			}
			if tc.mock.latestCalled {
				r.EXPECT().GetLatestDryrunByTarget(context.Background(), testTargetID).Return(tc.mock.latest, tc.mock.latestErr)
			}
			if tc.mock.updateCalled {
				r.EXPECT().UpdateTarget(context.Background(), tc.mock.updateInput).Return(tc.mock.update, tc.mock.updateErr)
			}
			if tc.mock.markCalled {
				r.EXPECT().DetailDataSource(context.Background(), testSourceID).Return(tc.mock.markDetail, tc.mock.markDetailErr)
				if tc.mock.markDetail.ID != "" {
					r.EXPECT().UpdateDataSource(context.Background(), repo.UpdateDataSourceOptions{ID: testSourceID, Status: string(model.SourceStatusPending), DryrunStatus: string(model.DryrunStatusPending)}).Return(testSource(model.SourceStatusPending), tc.mock.markUpdateErr)
				}
			}

			output, err := uc.UpdateTarget(context.Background(), tc.input)

			require.ErrorIs(t, err, tc.err)
			require.Equal(t, tc.output, output)
		})
	}
}

func TestActivateTarget(t *testing.T) {
	type mockData struct {
		target      model.CrawlTarget
		targetErr   error
		source      model.DataSource
		sourceErr   error
		guardLatest model.DryrunResult
		guardErr    error
		latest      model.DryrunResult
		latestErr   error
		update      model.CrawlTarget
		updateErr   error
		readyErr    error
	}
	tcs := map[string]struct {
		input  datasource.ActivateTargetInput
		mock   mockData
		output datasource.ActivateTargetOutput
		err    error
	}{
		"success": {input: datasource.ActivateTargetInput{DataSourceID: testSourceID, ID: testTargetID}, mock: mockData{target: testCrawlTarget(false), source: testSource(model.SourceStatusReady), guardLatest: model.DryrunResult{ID: "dryrun-1", Status: model.DryrunStatusSuccess}, latest: model.DryrunResult{ID: "dryrun-1", Status: model.DryrunStatusSuccess}, update: testCrawlTarget(true)}, output: datasource.ActivateTargetOutput{Target: testCrawlTarget(true)}},
		"invalid": {err: datasource.ErrTargetNotFound},
		"target_not_found": {
			input: datasource.ActivateTargetInput{DataSourceID: testSourceID, ID: testTargetID},
			mock:  mockData{targetErr: repo.ErrTargetNotFound},
			err:   datasource.ErrTargetNotFound,
		},
		"target_repo_error": {
			input: datasource.ActivateTargetInput{DataSourceID: testSourceID, ID: testTargetID},
			mock:  mockData{targetErr: errors.New("db")},
			err:   datasource.ErrTargetUpdateFailed,
		},
		"source_error": {
			input: datasource.ActivateTargetInput{DataSourceID: testSourceID, ID: testTargetID},
			mock:  mockData{target: testCrawlTarget(false), sourceErr: errors.New("db")},
			err:   datasource.ErrTargetUpdateFailed,
		},
		"dryrun_running": {
			input: datasource.ActivateTargetInput{DataSourceID: testSourceID, ID: testTargetID},
			mock:  mockData{target: testCrawlTarget(false), source: testSource(model.SourceStatusReady), guardLatest: model.DryrunResult{ID: "dryrun-1", Status: model.DryrunStatusRunning}},
			err:   datasource.ErrTargetDryrunRunning,
		},
		"latest_error": {
			input: datasource.ActivateTargetInput{DataSourceID: testSourceID, ID: testTargetID},
			mock:  mockData{target: testCrawlTarget(false), source: testSource(model.SourceStatusReady), guardLatest: model.DryrunResult{}, latestErr: errors.New("db")},
			err:   datasource.ErrTargetUpdateFailed,
		},
		"latest_not_usable": {
			input: datasource.ActivateTargetInput{DataSourceID: testSourceID, ID: testTargetID},
			mock:  mockData{target: testCrawlTarget(false), source: testSource(model.SourceStatusReady), guardLatest: model.DryrunResult{}, latest: model.DryrunResult{ID: "dryrun-1", Status: model.DryrunStatusFailed}},
			err:   datasource.ErrTargetActivateNotAllowed,
		},
		"update_not_found": {
			input: datasource.ActivateTargetInput{DataSourceID: testSourceID, ID: testTargetID},
			mock:  mockData{target: testCrawlTarget(false), source: testSource(model.SourceStatusReady), guardLatest: model.DryrunResult{}, latest: model.DryrunResult{ID: "dryrun-1", Status: model.DryrunStatusSuccess}, updateErr: repo.ErrTargetNotFound},
			err:   datasource.ErrTargetNotFound,
		},
		"update_error": {
			input: datasource.ActivateTargetInput{DataSourceID: testSourceID, ID: testTargetID},
			mock:  mockData{target: testCrawlTarget(false), source: testSource(model.SourceStatusReady), guardLatest: model.DryrunResult{}, latest: model.DryrunResult{ID: "dryrun-1", Status: model.DryrunStatusSuccess}, updateErr: errors.New("db")},
			err:   datasource.ErrTargetUpdateFailed,
		},
		"already_active": {input: datasource.ActivateTargetInput{DataSourceID: testSourceID, ID: testTargetID}, mock: mockData{target: testCrawlTarget(true), source: testSource(model.SourceStatusReady), guardLatest: model.DryrunResult{}}, output: datasource.ActivateTargetOutput{Target: testCrawlTarget(true)}},
		"no_dryrun_required_promote_source": {
			input: datasource.ActivateTargetInput{DataSourceID: testSourceID, ID: testTargetID},
			mock: mockData{target: func() model.CrawlTarget {
				target := testCrawlTarget(false)
				target.TargetType = model.TargetTypeProfile
				return target
			}(), source: testSource(model.SourceStatusPending), update: testCrawlTarget(true)},
			output: datasource.ActivateTargetOutput{Target: testCrawlTarget(true)},
		},
		"no_dryrun_required_promote_source_error": {
			input: datasource.ActivateTargetInput{DataSourceID: testSourceID, ID: testTargetID},
			mock: mockData{target: func() model.CrawlTarget {
				target := testCrawlTarget(false)
				target.TargetType = model.TargetTypeProfile
				return target
			}(), source: testSource(model.SourceStatusPending), update: testCrawlTarget(true), readyErr: errors.New("db")},
			output: datasource.ActivateTargetOutput{Target: testCrawlTarget(true)},
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			uc, r, _ := newDatasourceUC(t)
			if tc.input.ID != "" {
				r.EXPECT().GetTarget(context.Background(), repo.GetTargetOptions{DataSourceID: testSourceID, ID: testTargetID}).Return(tc.mock.target, tc.mock.targetErr)
				if tc.mock.targetErr == nil {
					r.EXPECT().DetailDataSource(context.Background(), testSourceID).Return(tc.mock.source, tc.mock.sourceErr)
				}
				dryrunNeeded := tc.mock.sourceErr == nil && model.IsDryrunRequired(tc.mock.source.SourceType, tc.mock.target.TargetType)
				if dryrunNeeded {
					r.EXPECT().GetLatestDryrunByTarget(context.Background(), testTargetID).Return(tc.mock.guardLatest, tc.mock.guardErr).Once()
				}
				if dryrunNeeded && !tc.mock.target.IsActive && tc.mock.guardErr == nil && tc.mock.guardLatest.Status != model.DryrunStatusRunning && (tc.err == datasource.ErrTargetUpdateFailed || tc.err == datasource.ErrTargetActivateNotAllowed || tc.err == datasource.ErrTargetNotFound || tc.err == nil) {
					r.EXPECT().GetLatestDryrunByTarget(context.Background(), testTargetID).Return(tc.mock.latest, tc.mock.latestErr).Once()
				}
				if !tc.mock.target.IsActive && (tc.err == nil || tc.mock.updateErr != nil) {
					r.EXPECT().UpdateTarget(context.Background(), repo.UpdateTargetOptions{DataSourceID: testSourceID, ID: testTargetID, IsActive: boolPtr(true)}).Return(tc.mock.update, tc.mock.updateErr)
				}
				if name == "no_dryrun_required_promote_source" || name == "no_dryrun_required_promote_source_error" {
					r.EXPECT().UpdateDataSource(context.Background(), repo.UpdateDataSourceOptions{ID: testSourceID, Status: string(model.SourceStatusReady), DryrunStatus: string(model.DryrunStatusNotRequired)}).Return(testSource(model.SourceStatusReady), tc.mock.readyErr)
				}
			}

			output, err := uc.ActivateTarget(context.Background(), tc.input)

			require.ErrorIs(t, err, tc.err)
			require.Equal(t, tc.output, output)
		})
	}
}

func TestDeactivateTarget(t *testing.T) {
	type mockData struct {
		target    model.CrawlTarget
		targetErr error
		latest    model.DryrunResult
		latestErr error
		source    model.DataSource
		count     int64
		countErr  error
		update    model.CrawlTarget
		updateErr error
	}
	tcs := map[string]struct {
		input  datasource.DeactivateTargetInput
		mock   mockData
		output datasource.DeactivateTargetOutput
		err    error
	}{
		"success":          {input: datasource.DeactivateTargetInput{DataSourceID: testSourceID, ID: testTargetID}, mock: mockData{target: testCrawlTarget(true), source: testSource(model.SourceStatusActive), count: 2, update: testCrawlTarget(false)}, output: datasource.DeactivateTargetOutput{Target: testCrawlTarget(false)}},
		"invalid":          {err: datasource.ErrTargetNotFound},
		"target_not_found": {input: datasource.DeactivateTargetInput{DataSourceID: testSourceID, ID: testTargetID}, mock: mockData{targetErr: repo.ErrTargetNotFound}, err: datasource.ErrTargetNotFound},
		"target_error":     {input: datasource.DeactivateTargetInput{DataSourceID: testSourceID, ID: testTargetID}, mock: mockData{targetErr: errors.New("db")}, err: datasource.ErrTargetUpdateFailed},
		"dryrun_error":     {input: datasource.DeactivateTargetInput{DataSourceID: testSourceID, ID: testTargetID}, mock: mockData{target: testCrawlTarget(true), latestErr: errors.New("db")}, err: datasource.ErrUpdateFailed},
		"dryrun_running":   {input: datasource.DeactivateTargetInput{DataSourceID: testSourceID, ID: testTargetID}, mock: mockData{target: testCrawlTarget(true), latest: model.DryrunResult{ID: "dryrun-1", Status: model.DryrunStatusRunning}}, err: datasource.ErrTargetDryrunRunning},
		"already_inactive": {input: datasource.DeactivateTargetInput{DataSourceID: testSourceID, ID: testTargetID}, mock: mockData{target: testCrawlTarget(false)}, output: datasource.DeactivateTargetOutput{Target: testCrawlTarget(false)}},
		"last_active":      {input: datasource.DeactivateTargetInput{DataSourceID: testSourceID, ID: testTargetID}, mock: mockData{target: testCrawlTarget(true), source: testSource(model.SourceStatusActive), count: 1}, err: datasource.ErrTargetDeactivateNotAllowed},
		"count_error":      {input: datasource.DeactivateTargetInput{DataSourceID: testSourceID, ID: testTargetID}, mock: mockData{target: testCrawlTarget(true), source: testSource(model.SourceStatusActive), countErr: errors.New("db")}, err: datasource.ErrTargetUpdateFailed},
		"update_not_found": {input: datasource.DeactivateTargetInput{DataSourceID: testSourceID, ID: testTargetID}, mock: mockData{target: testCrawlTarget(true), source: testSource(model.SourceStatusActive), count: 2, updateErr: repo.ErrTargetNotFound}, err: datasource.ErrTargetNotFound},
		"update_error":     {input: datasource.DeactivateTargetInput{DataSourceID: testSourceID, ID: testTargetID}, mock: mockData{target: testCrawlTarget(true), source: testSource(model.SourceStatusActive), count: 2, updateErr: errors.New("db")}, err: datasource.ErrTargetUpdateFailed},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			uc, r, _ := newDatasourceUC(t)
			if tc.input.ID != "" {
				r.EXPECT().GetTarget(context.Background(), repo.GetTargetOptions{DataSourceID: testSourceID, ID: testTargetID}).Return(tc.mock.target, tc.mock.targetErr)
				if tc.mock.targetErr == nil {
					r.EXPECT().GetLatestDryrunByTarget(context.Background(), testTargetID).Return(tc.mock.latest, tc.mock.latestErr)
				}
				if tc.mock.targetErr == nil && tc.mock.latestErr == nil && tc.mock.latest.Status != model.DryrunStatusRunning && tc.mock.target.IsActive {
					r.EXPECT().DetailDataSource(context.Background(), testSourceID).Return(tc.mock.source, nil)
					if tc.mock.source.Status == model.SourceStatusActive {
						r.EXPECT().CountActiveTargets(context.Background(), testSourceID).Return(tc.mock.count, tc.mock.countErr)
					}
				}
				if tc.mock.target.IsActive && (tc.err == nil || tc.mock.updateErr != nil) {
					r.EXPECT().UpdateTarget(context.Background(), repo.UpdateTargetOptions{DataSourceID: testSourceID, ID: testTargetID, IsActive: boolPtr(false)}).Return(tc.mock.update, tc.mock.updateErr)
				}
			}

			output, err := uc.DeactivateTarget(context.Background(), tc.input)

			require.ErrorIs(t, err, tc.err)
			require.Equal(t, tc.output, output)
		})
	}
}

func TestDeleteTarget(t *testing.T) {
	type mockData struct {
		target    model.CrawlTarget
		targetErr error
		latest    model.DryrunResult
		latestErr error
		source    model.DataSource
		count     int64
		countErr  error
		deleteErr error
	}
	tcs := map[string]struct {
		input  datasource.DeleteTargetInput
		mock   mockData
		output struct{}
		err    error
	}{
		"success":          {input: datasource.DeleteTargetInput{DataSourceID: testSourceID, ID: testTargetID}, mock: mockData{target: testCrawlTarget(false)}},
		"invalid":          {err: datasource.ErrTargetNotFound},
		"target_not_found": {input: datasource.DeleteTargetInput{DataSourceID: testSourceID, ID: testTargetID}, mock: mockData{targetErr: repo.ErrTargetNotFound}, err: datasource.ErrTargetNotFound},
		"target_error":     {input: datasource.DeleteTargetInput{DataSourceID: testSourceID, ID: testTargetID}, mock: mockData{targetErr: errors.New("db")}, err: datasource.ErrTargetDeleteFailed},
		"dryrun_error":     {input: datasource.DeleteTargetInput{DataSourceID: testSourceID, ID: testTargetID}, mock: mockData{target: testCrawlTarget(false), latestErr: errors.New("db")}, err: datasource.ErrUpdateFailed},
		"dryrun_running":   {input: datasource.DeleteTargetInput{DataSourceID: testSourceID, ID: testTargetID}, mock: mockData{target: testCrawlTarget(false), latest: model.DryrunResult{ID: "dryrun-1", Status: model.DryrunStatusRunning}}, err: datasource.ErrTargetDryrunRunning},
		"last_active":      {input: datasource.DeleteTargetInput{DataSourceID: testSourceID, ID: testTargetID}, mock: mockData{target: testCrawlTarget(true), source: testSource(model.SourceStatusActive), count: 1}, err: datasource.ErrTargetDeleteNotAllowed},
		"count_error":      {input: datasource.DeleteTargetInput{DataSourceID: testSourceID, ID: testTargetID}, mock: mockData{target: testCrawlTarget(true), source: testSource(model.SourceStatusActive), countErr: errors.New("db")}, err: datasource.ErrTargetDeleteFailed},
		"delete_not_found": {input: datasource.DeleteTargetInput{DataSourceID: testSourceID, ID: testTargetID}, mock: mockData{target: testCrawlTarget(false), deleteErr: repo.ErrTargetNotFound}, err: datasource.ErrTargetNotFound},
		"delete_error":     {input: datasource.DeleteTargetInput{DataSourceID: testSourceID, ID: testTargetID}, mock: mockData{target: testCrawlTarget(false), deleteErr: errors.New("db")}, err: datasource.ErrTargetDeleteFailed},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			uc, r, _ := newDatasourceUC(t)
			if tc.input.ID != "" {
				r.EXPECT().GetTarget(context.Background(), repo.GetTargetOptions{DataSourceID: testSourceID, ID: testTargetID}).Return(tc.mock.target, tc.mock.targetErr)
				if tc.mock.targetErr == nil {
					r.EXPECT().GetLatestDryrunByTarget(context.Background(), testTargetID).Return(tc.mock.latest, tc.mock.latestErr)
				}
				if tc.mock.targetErr == nil && tc.mock.latestErr == nil && tc.mock.latest.Status != model.DryrunStatusRunning && tc.mock.target.IsActive {
					r.EXPECT().DetailDataSource(context.Background(), testSourceID).Return(tc.mock.source, nil)
					if tc.mock.source.Status == model.SourceStatusActive {
						r.EXPECT().CountActiveTargets(context.Background(), testSourceID).Return(tc.mock.count, tc.mock.countErr)
					}
				}
				if tc.err == nil || tc.mock.deleteErr != nil {
					r.EXPECT().DeleteTarget(context.Background(), repo.DeleteTargetOptions{DataSourceID: testSourceID, ID: testTargetID}).Return(tc.mock.deleteErr)
				}
			}

			err := uc.DeleteTarget(context.Background(), tc.input)

			require.ErrorIs(t, err, tc.err)
		})
	}
}

func TestGetActivationReadiness(t *testing.T) {
	type mockData struct {
		sources   []model.DataSource
		sourceErr error
		targets   []model.CrawlTarget
		targetErr error
		latest    model.DryrunResult
		latestErr error
	}
	tcs := map[string]struct {
		input  datasource.ActivationReadinessInput
		mock   mockData
		output bool
		err    error
	}{
		"success": {input: datasource.ActivationReadinessInput{ProjectID: testProjectID}, mock: mockData{sources: []model.DataSource{testSource(model.SourceStatusReady)}, targets: []model.CrawlTarget{testCrawlTarget(true)}, latest: model.DryrunResult{ID: "dryrun-1", Status: model.DryrunStatusSuccess}}, output: true},
		"resume_success": {
			input:  datasource.ActivationReadinessInput{ProjectID: testProjectID, Command: datasource.ActivationReadinessCommandResume},
			mock:   mockData{sources: []model.DataSource{testSource(model.SourceStatusPaused)}, targets: []model.CrawlTarget{testCrawlTarget(true)}, latest: model.DryrunResult{ID: "dryrun-1", Status: model.DryrunStatusSuccess}},
			output: true,
		},
		"invalid":           {input: datasource.ActivationReadinessInput{}, err: datasource.ErrProjectIDRequired},
		"invalid_command":   {input: datasource.ActivationReadinessInput{ProjectID: testProjectID, Command: "bad"}, err: datasource.ErrInvalidReadinessCommand},
		"empty":             {input: datasource.ActivationReadinessInput{ProjectID: testProjectID}, output: false},
		"list_error":        {input: datasource.ActivationReadinessInput{ProjectID: testProjectID}, mock: mockData{sourceErr: errors.New("db")}, err: datasource.ErrListFailed},
		"target_list_error": {input: datasource.ActivationReadinessInput{ProjectID: testProjectID}, mock: mockData{sources: []model.DataSource{testSource(model.SourceStatusReady)}, targetErr: errors.New("db")}, err: datasource.ErrListFailed},
		"latest_error":      {input: datasource.ActivationReadinessInput{ProjectID: testProjectID}, mock: mockData{sources: []model.DataSource{testSource(model.SourceStatusReady)}, targets: []model.CrawlTarget{testCrawlTarget(true)}, latestErr: errors.New("db")}, err: datasource.ErrListFailed},
		"missing_dryrun":    {input: datasource.ActivationReadinessInput{ProjectID: testProjectID}, mock: mockData{sources: []model.DataSource{testSource(model.SourceStatusReady)}, targets: []model.CrawlTarget{testCrawlTarget(true)}}, output: false},
		"failed_dryrun":     {input: datasource.ActivationReadinessInput{ProjectID: testProjectID}, mock: mockData{sources: []model.DataSource{testSource(model.SourceStatusReady)}, targets: []model.CrawlTarget{testCrawlTarget(true)}, latest: model.DryrunResult{ID: "dryrun-1", Status: model.DryrunStatusFailed}}, output: false},
		"passive_unconfirmed": {
			input: datasource.ActivationReadinessInput{ProjectID: testProjectID},
			mock: mockData{sources: []model.DataSource{func() model.DataSource {
				source := testSource(model.SourceStatusReady)
				source.SourceCategory = model.SourceCategoryPassive
				source.OnboardingStatus = model.OnboardingStatusPending
				return source
			}()}},
			output: false,
		},
		"unknown_category_skipped": {
			input: datasource.ActivationReadinessInput{ProjectID: testProjectID},
			mock: mockData{sources: []model.DataSource{func() model.DataSource {
				source := testSource(model.SourceStatusReady)
				source.SourceCategory = model.SourceCategory("OTHER")
				return source
			}()}},
			output: true,
		},
		"invalid_status": {
			input:  datasource.ActivationReadinessInput{ProjectID: testProjectID},
			mock:   mockData{sources: []model.DataSource{testSource(model.SourceStatusPending)}, targets: []model.CrawlTarget{testCrawlTarget(true)}, latest: model.DryrunResult{ID: "dryrun-1", Status: model.DryrunStatusSuccess}},
			output: false,
		},
		"no_active_target": {
			input: datasource.ActivationReadinessInput{ProjectID: testProjectID},
			mock: mockData{sources: []model.DataSource{testSource(model.SourceStatusReady)}, targets: []model.CrawlTarget{func() model.CrawlTarget {
				target := testCrawlTarget(false)
				return target
			}()}, latest: model.DryrunResult{ID: "dryrun-1", Status: model.DryrunStatusSuccess}},
			output: false,
		},
		"no_dryrun_required_target": {
			input: datasource.ActivationReadinessInput{ProjectID: testProjectID},
			mock: mockData{sources: []model.DataSource{testSource(model.SourceStatusReady)}, targets: []model.CrawlTarget{func() model.CrawlTarget {
				target := testCrawlTarget(true)
				target.TargetType = model.TargetTypeProfile
				return target
			}()}},
			output: true,
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			uc, r, _ := newDatasourceUC(t)
			if tc.input.ProjectID != "" && tc.err != datasource.ErrInvalidReadinessCommand {
				r.EXPECT().ListDataSources(context.Background(), repo.ListDataSourcesOptions{ProjectID: testProjectID}).Return(tc.mock.sources, tc.mock.sourceErr)
				if len(tc.mock.sources) > 0 && tc.mock.sources[0].SourceCategory == model.SourceCategoryCrawl && tc.mock.sourceErr == nil {
					r.EXPECT().ListTargets(context.Background(), repo.ListTargetsOptions{DataSourceID: testSourceID}).Return(tc.mock.targets, tc.mock.targetErr)
					if len(tc.mock.targets) > 0 && tc.mock.targetErr == nil {
						if model.IsDryrunRequired(tc.mock.sources[0].SourceType, tc.mock.targets[0].TargetType) {
							r.EXPECT().GetLatestDryrunByTarget(context.Background(), testTargetID).Return(tc.mock.latest, tc.mock.latestErr)
						}
					}
				}
			}

			output, err := uc.GetActivationReadiness(context.Background(), tc.input)

			require.ErrorIs(t, err, tc.err)
			if tc.err == nil {
				require.Equal(t, tc.output, output.CanProceed)
			}
		})
	}
}

func TestActivate(t *testing.T) {
	type mockData struct {
		readinessSources []model.DataSource
		readinessErr     error
		lifecycleSources []model.DataSource
		lifecycleErr     error
		updateErr        error
	}
	tcs := map[string]struct {
		input  string
		mock   mockData
		output datasource.ProjectLifecycleOutput
		err    error
	}{
		"success": {input: testProjectID, mock: mockData{readinessSources: []model.DataSource{testSource(model.SourceStatusReady)}, lifecycleSources: []model.DataSource{testSource(model.SourceStatusReady)}}, output: datasource.ProjectLifecycleOutput{ProjectID: testProjectID, AffectedDataSourceCount: 1}},
		"invalid": {err: datasource.ErrProjectIDRequired},
		"readiness_blocked": {
			input: testProjectID,
			mock:  mockData{},
			err:   datasource.ErrActivationReadinessFailed,
		},
		"readiness_error": {
			input: testProjectID,
			mock:  mockData{readinessErr: errors.New("db")},
			err:   datasource.ErrListFailed,
		},
		"lifecycle_list_error": {
			input: testProjectID,
			mock:  mockData{readinessSources: []model.DataSource{testSource(model.SourceStatusReady)}, lifecycleErr: errors.New("db")},
			err:   datasource.ErrListFailed,
		},
		"lifecycle_not_eligible": {
			input: testProjectID,
			mock:  mockData{readinessSources: []model.DataSource{testSource(model.SourceStatusReady)}, lifecycleSources: []model.DataSource{testSource(model.SourceStatusPending)}},
			err:   datasource.ErrActivateNotAllowed,
		},
		"update_error": {
			input: testProjectID,
			mock:  mockData{readinessSources: []model.DataSource{testSource(model.SourceStatusReady)}, lifecycleSources: []model.DataSource{testSource(model.SourceStatusReady)}, updateErr: errors.New("db")},
			err:   datasource.ErrUpdateFailed,
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			uc, r, _ := newDatasourceUC(t)
			if tc.input != "" {
				r.EXPECT().ListDataSources(context.Background(), repo.ListDataSourcesOptions{ProjectID: testProjectID}).Return(tc.mock.readinessSources, tc.mock.readinessErr).Once()
				if tc.mock.readinessErr == nil && len(tc.mock.readinessSources) > 0 {
					r.EXPECT().ListTargets(context.Background(), repo.ListTargetsOptions{DataSourceID: testSourceID}).Return([]model.CrawlTarget{testCrawlTarget(true)}, nil)
					r.EXPECT().GetLatestDryrunByTarget(context.Background(), testTargetID).Return(model.DryrunResult{ID: "dryrun-1", Status: model.DryrunStatusSuccess}, nil)
					r.EXPECT().ListDataSources(context.Background(), repo.ListDataSourcesOptions{ProjectID: testProjectID}).Return(tc.mock.lifecycleSources, tc.mock.lifecycleErr).Once()
				}
				if tc.err == nil || tc.mock.updateErr != nil {
					r.EXPECT().UpdateProjectDataSourcesLifecycle(context.Background(), uc.buildProjectLifecycleUpdateOptions(testProjectID, "activate")).Return(int64(1), tc.mock.updateErr)
				}
			}

			output, err := uc.Activate(context.Background(), tc.input)

			require.ErrorIs(t, err, tc.err)
			require.Equal(t, tc.output, output)
		})
	}
}

func TestPause(t *testing.T) {
	type mockData struct {
		updateErr error
		cancelErr error
	}
	tcs := map[string]struct {
		input  string
		mock   mockData
		output datasource.ProjectLifecycleOutput
		err    error
	}{
		"success": {input: testProjectID, output: datasource.ProjectLifecycleOutput{ProjectID: testProjectID, AffectedDataSourceCount: 1}},
		"invalid": {err: datasource.ErrProjectIDRequired},
		"update_error": {
			input: testProjectID,
			mock:  mockData{updateErr: errors.New("db")},
			err:   datasource.ErrUpdateFailed,
		},
		"cancel_error": {
			input: testProjectID,
			mock:  mockData{cancelErr: errors.New("cancel")},
			err:   datasource.ErrUpdateFailed,
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			uc, r, _ := newDatasourceUC(t)
			if tc.input != "" {
				r.EXPECT().UpdateProjectDataSourcesLifecycle(context.Background(), uc.buildProjectLifecycleUpdateOptions(testProjectID, "pause")).Return(int64(1), tc.mock.updateErr)
				if tc.mock.updateErr == nil && tc.mock.cancelErr != nil {
					execUC := execution.NewMockUseCase(t)
					uc.exec = execUC
					execUC.EXPECT().CancelProjectRuntime(context.Background(), mock.MatchedBy(func(input execution.CancelProjectRuntimeInput) bool {
						return input.ProjectID == testProjectID && input.Reason == "cancelled due to project pause"
					})).Return(tc.mock.cancelErr)
				}
			}

			output, err := uc.Pause(context.Background(), tc.input)

			require.ErrorIs(t, err, tc.err)
			require.Equal(t, tc.output, output)
		})
	}
}

func TestResume(t *testing.T) {
	type mockData struct {
		readinessSources []model.DataSource
		readinessErr     error
		lifecycleSources []model.DataSource
		lifecycleErr     error
		updateErr        error
	}
	tcs := map[string]struct {
		input  string
		mock   mockData
		output datasource.ProjectLifecycleOutput
		err    error
	}{
		"success": {input: testProjectID, mock: mockData{readinessSources: []model.DataSource{testSource(model.SourceStatusPaused)}, lifecycleSources: []model.DataSource{testSource(model.SourceStatusPaused)}}, output: datasource.ProjectLifecycleOutput{ProjectID: testProjectID, AffectedDataSourceCount: 1}},
		"invalid": {err: datasource.ErrProjectIDRequired},
		"readiness_blocked": {
			input: testProjectID,
			mock:  mockData{},
			err:   datasource.ErrActivationReadinessFailed,
		},
		"readiness_error": {
			input: testProjectID,
			mock:  mockData{readinessErr: errors.New("db")},
			err:   datasource.ErrListFailed,
		},
		"lifecycle_list_error": {
			input: testProjectID,
			mock:  mockData{readinessSources: []model.DataSource{testSource(model.SourceStatusPaused)}, lifecycleErr: errors.New("db")},
			err:   datasource.ErrListFailed,
		},
		"lifecycle_not_eligible": {
			input: testProjectID,
			mock:  mockData{readinessSources: []model.DataSource{testSource(model.SourceStatusPaused)}, lifecycleSources: []model.DataSource{testSource(model.SourceStatusReady)}},
			err:   datasource.ErrResumeNotAllowed,
		},
		"update_error": {
			input: testProjectID,
			mock:  mockData{readinessSources: []model.DataSource{testSource(model.SourceStatusPaused)}, lifecycleSources: []model.DataSource{testSource(model.SourceStatusPaused)}, updateErr: errors.New("db")},
			err:   datasource.ErrUpdateFailed,
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			uc, r, _ := newDatasourceUC(t)
			if tc.input != "" {
				r.EXPECT().ListDataSources(context.Background(), repo.ListDataSourcesOptions{ProjectID: testProjectID}).Return(tc.mock.readinessSources, tc.mock.readinessErr).Once()
				if tc.mock.readinessErr == nil && len(tc.mock.readinessSources) > 0 {
					r.EXPECT().ListTargets(context.Background(), repo.ListTargetsOptions{DataSourceID: testSourceID}).Return([]model.CrawlTarget{testCrawlTarget(true)}, nil)
					r.EXPECT().GetLatestDryrunByTarget(context.Background(), testTargetID).Return(model.DryrunResult{ID: "dryrun-1", Status: model.DryrunStatusSuccess}, nil)
					r.EXPECT().ListDataSources(context.Background(), repo.ListDataSourcesOptions{ProjectID: testProjectID}).Return(tc.mock.lifecycleSources, tc.mock.lifecycleErr).Once()
				}
				if tc.err == nil || tc.mock.updateErr != nil {
					r.EXPECT().UpdateProjectDataSourcesLifecycle(context.Background(), uc.buildProjectLifecycleUpdateOptions(testProjectID, "resume")).Return(int64(1), tc.mock.updateErr)
				}
			}

			output, err := uc.Resume(context.Background(), tc.input)

			require.ErrorIs(t, err, tc.err)
			require.Equal(t, tc.output, output)
		})
	}
}

func boolPtr(v bool) *bool {
	return &v
}
