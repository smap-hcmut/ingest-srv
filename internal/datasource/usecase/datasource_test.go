package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"ingest-srv/internal/datasource"
	repo "ingest-srv/internal/datasource/repository"
	"ingest-srv/internal/model"
	"ingest-srv/pkg/microservice"

	"github.com/aarondl/sqlboiler/v4/types"
	"github.com/smap-hcmut/shared-libs/go/log"
	"github.com/smap-hcmut/shared-libs/go/paginator"
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
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			uc, r, _ := newDatasourceUC(t)
			if tc.mock.detailCalled {
				r.EXPECT().DetailDataSource(context.Background(), testSourceID).Return(tc.mock.detailOutput, tc.mock.detailErr)
			}
			if tc.mock.targetCalled {
				r.EXPECT().ListTargets(context.Background(), repo.ListTargetsOptions{DataSourceID: testSourceID}).Return(tc.mock.targetOutput, nil)
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
		archiveCalled bool
		archiveErr    error
	}

	tcs := map[string]struct {
		input  string
		mock   mockArchive
		output struct{}
		err    error
	}{
		"success":          {input: testSourceID, mock: mockArchive{detailCalled: true, detailOutput: testSource(model.SourceStatusReady), targetCalled: true, archiveCalled: true}},
		"invalid_input":    {err: datasource.ErrNotFound},
		"already_archived": {input: testSourceID, mock: mockArchive{detailCalled: true, detailOutput: testSource(model.SourceStatusArchived), targetCalled: true}},
		"repo_error":       {input: testSourceID, mock: mockArchive{detailCalled: true, detailOutput: testSource(model.SourceStatusReady), targetCalled: true, archiveCalled: true, archiveErr: errors.New("db")}, err: datasource.ErrDeleteFailed},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			uc, r, _ := newDatasourceUC(t)
			if tc.mock.detailCalled {
				r.EXPECT().DetailDataSource(context.Background(), tc.input).Return(tc.mock.detailOutput, tc.mock.detailErr)
			}
			if tc.mock.targetCalled {
				r.EXPECT().ListTargets(context.Background(), repo.ListTargetsOptions{DataSourceID: tc.mock.detailOutput.ID}).Return(nil, nil)
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
		targetCalled bool
		deleteCalled bool
		deleteErr    error
	}

	tcs := map[string]struct {
		input  string
		mock   mockDelete
		output struct{}
		err    error
	}{
		"success":           {input: testSourceID, mock: mockDelete{detailCalled: true, detailOutput: testSource(model.SourceStatusArchived), targetCalled: true, deleteCalled: true}},
		"invalid_input":     {err: datasource.ErrNotFound},
		"requires_archived": {input: testSourceID, mock: mockDelete{detailCalled: true, detailOutput: testSource(model.SourceStatusReady), targetCalled: true}, err: datasource.ErrDeleteRequiresArchived},
		"delete_repo_error": {input: testSourceID, mock: mockDelete{detailCalled: true, detailOutput: testSource(model.SourceStatusArchived), targetCalled: true, deleteCalled: true, deleteErr: errors.New("db")}, err: datasource.ErrDeleteFailed},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			uc, r, _ := newDatasourceUC(t)
			if tc.mock.detailCalled {
				r.EXPECT().DetailDataSource(context.Background(), tc.input).Return(tc.mock.detailOutput, nil)
			}
			if tc.mock.targetCalled {
				r.EXPECT().ListTargets(context.Background(), repo.ListTargetsOptions{DataSourceID: tc.mock.detailOutput.ID}).Return(nil, nil)
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
	}

	tcs := map[string]struct {
		input  datasource.MarkDryrunRunningInput
		mock   mockData
		output datasource.MarkDryrunRunningOutput
		err    error
	}{
		"success":   {input: datasource.MarkDryrunRunningInput{ID: testSourceID, DryrunLastResultID: "result-1"}, mock: mockData{detail: testSource(model.SourceStatusReady), update: repo.UpdateDataSourceOptions{ID: testSourceID, DryrunStatus: string(model.DryrunStatusRunning), DryrunLastResultID: "result-1", Status: string(model.SourceStatusPending)}, output: testSource(model.SourceStatusPending)}, output: datasource.MarkDryrunRunningOutput{DataSource: testSource(model.SourceStatusPending)}},
		"not_found": {input: datasource.MarkDryrunRunningInput{ID: testSourceID}, mock: mockData{err: errors.New("db")}, err: datasource.ErrUpdateFailed},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			uc, r, _ := newDatasourceUC(t)
			r.EXPECT().DetailDataSource(context.Background(), tc.input.ID).Return(tc.mock.detail, tc.mock.err)
			if tc.err == nil {
				r.EXPECT().UpdateDataSource(context.Background(), tc.mock.update).Return(tc.mock.output, nil)
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
		update repo.UpdateDataSourceOptions
		output model.DataSource
	}

	tcs := map[string]struct {
		input  datasource.ApplyDryrunResultInput
		mock   mockData
		output datasource.ApplyDryrunResultOutput
		err    error
	}{
		"success_failed":  {input: datasource.ApplyDryrunResultInput{ID: testSourceID, DryrunLastResultID: "result-1", DryrunStatus: string(model.DryrunStatusFailed)}, mock: mockData{detail: testSource(model.SourceStatusReady), update: repo.UpdateDataSourceOptions{ID: testSourceID, DryrunStatus: string(model.DryrunStatusFailed), DryrunLastResultID: "result-1", Status: string(model.SourceStatusPending)}, output: testSource(model.SourceStatusPending)}, output: datasource.ApplyDryrunResultOutput{DataSource: testSource(model.SourceStatusPending)}},
		"success_success": {input: datasource.ApplyDryrunResultInput{ID: testSourceID, DryrunLastResultID: "result-1", DryrunStatus: string(model.DryrunStatusSuccess)}, mock: mockData{detail: testSource(model.SourceStatusPending), update: repo.UpdateDataSourceOptions{ID: testSourceID, DryrunStatus: string(model.DryrunStatusSuccess), DryrunLastResultID: "result-1", Status: string(model.SourceStatusReady)}, output: testSource(model.SourceStatusReady)}, output: datasource.ApplyDryrunResultOutput{DataSource: testSource(model.SourceStatusReady)}},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			uc, r, _ := newDatasourceUC(t)
			r.EXPECT().DetailDataSource(context.Background(), tc.input.ID).Return(tc.mock.detail, nil)
			r.EXPECT().UpdateDataSource(context.Background(), tc.mock.update).Return(tc.mock.output, nil)

			output, err := uc.ApplyDryrunResult(context.Background(), tc.input)

			require.NoError(t, err)
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
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			uc, r, _ := newDatasourceUC(t)
			if tc.mock.detailCalled {
				r.EXPECT().DetailDataSource(context.Background(), testSourceID).Return(tc.mock.detailOutput, nil)
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
		detail  model.DataSource
		targets []model.CrawlTarget
		latest  model.DryrunResult
		update  model.DataSource
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
		"invalid":      {err: datasource.ErrNotFound},
		"wrong_status": {input: testSourceID, mock: mockData{detail: testSource(model.SourceStatusPending)}, err: datasource.ErrActivateNotAllowed},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			uc, r, _ := newDatasourceUC(t)
			if tc.input != "" {
				r.EXPECT().DetailDataSource(context.Background(), testSourceID).Return(tc.mock.detail, nil)
			}
			if tc.err == nil {
				r.EXPECT().ListTargets(context.Background(), repo.ListTargetsOptions{DataSourceID: testSourceID}).Return(tc.mock.targets, nil)
				r.EXPECT().GetLatestDryrunByTarget(context.Background(), testTargetID).Return(tc.mock.latest, nil)
				r.EXPECT().UpdateDataSource(context.Background(), repo.UpdateDataSourceOptions{ID: testSourceID, Status: string(model.SourceStatusActive), ClearPausedAt: true}).Return(tc.mock.update, nil)
			}

			output, err := uc.ActivateDataSource(context.Background(), tc.input)

			require.ErrorIs(t, err, tc.err)
			require.Equal(t, tc.output, output)
		})
	}
}

func TestPauseDataSource(t *testing.T) {
	tcs := map[string]struct {
		input  string
		mock   model.DataSource
		output datasource.PauseOutput
		err    error
	}{
		"success":      {input: testSourceID, mock: testSource(model.SourceStatusActive), output: datasource.PauseOutput{DataSource: testSource(model.SourceStatusPaused)}},
		"invalid":      {err: datasource.ErrNotFound},
		"wrong_status": {input: testSourceID, mock: testSource(model.SourceStatusReady), err: datasource.ErrPauseNotAllowed},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			uc, r, _ := newDatasourceUC(t)
			if tc.input != "" {
				r.EXPECT().DetailDataSource(context.Background(), testSourceID).Return(tc.mock, nil)
			}
			if tc.err == nil {
				r.EXPECT().UpdateDataSource(context.Background(), repo.UpdateDataSourceOptions{ID: testSourceID, Status: string(model.SourceStatusPaused)}).Return(testSource(model.SourceStatusPaused), nil)
			}

			output, err := uc.PauseDataSource(context.Background(), tc.input)

			require.ErrorIs(t, err, tc.err)
			require.Equal(t, tc.output, output)
		})
	}
}

func TestResumeDataSource(t *testing.T) {
	tcs := map[string]struct {
		input  string
		mock   model.DataSource
		output datasource.ResumeOutput
		err    error
	}{
		"success":      {input: testSourceID, mock: testSource(model.SourceStatusPaused), output: datasource.ResumeOutput{DataSource: testSource(model.SourceStatusActive)}},
		"invalid":      {err: datasource.ErrNotFound},
		"wrong_status": {input: testSourceID, mock: testSource(model.SourceStatusReady), err: datasource.ErrResumeNotAllowed},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			uc, r, _ := newDatasourceUC(t)
			if tc.input != "" {
				r.EXPECT().DetailDataSource(context.Background(), testSourceID).Return(tc.mock, nil)
			}
			if tc.err == nil {
				r.EXPECT().ListTargets(context.Background(), repo.ListTargetsOptions{DataSourceID: testSourceID}).Return([]model.CrawlTarget{testCrawlTarget(true)}, nil)
				r.EXPECT().GetLatestDryrunByTarget(context.Background(), testTargetID).Return(model.DryrunResult{ID: "dryrun-1", Status: model.DryrunStatusSuccess}, nil)
				r.EXPECT().UpdateDataSource(context.Background(), repo.UpdateDataSourceOptions{ID: testSourceID, Status: string(model.SourceStatusActive), ClearPausedAt: true}).Return(testSource(model.SourceStatusActive), nil)
			}

			output, err := uc.ResumeDataSource(context.Background(), tc.input)

			require.ErrorIs(t, err, tc.err)
			require.Equal(t, tc.output, output)
		})
	}
}

func TestUpdateCrawlMode(t *testing.T) {
	type mockData struct {
		detail model.DataSource
		update model.DataSource
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
		"invalid": {input: datasource.UpdateCrawlModeInput{ID: testSourceID, CrawlMode: "bad", TriggerType: string(model.TriggerTypeManual)}, err: datasource.ErrInvalidCrawlMode},
		"not_crawl": {input: datasource.UpdateCrawlModeInput{ID: testSourceID, CrawlMode: string(model.CrawlModeNormal), TriggerType: string(model.TriggerTypeManual)}, mock: mockData{detail: func() model.DataSource {
			ds := testSource(model.SourceStatusActive)
			ds.SourceCategory = model.SourceCategoryPassive
			return ds
		}()}, err: datasource.ErrCrawlModeNotAllowed},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			uc, r, _ := newDatasourceUC(t)
			if tc.err != datasource.ErrInvalidCrawlMode {
				r.EXPECT().DetailDataSource(context.Background(), testSourceID).Return(tc.mock.detail, nil)
			}
			if tc.err == nil {
				r.EXPECT().UpdateDataSource(context.Background(), repo.UpdateDataSourceOptions{ID: testSourceID, CrawlMode: string(model.CrawlModeCrisis)}).Return(tc.mock.update, nil)
				r.EXPECT().CreateCrawlModeChange(context.Background(), repo.CreateCrawlModeChangeOptions{SourceID: testSourceID, ProjectID: testProjectID, TriggerType: string(model.TriggerTypeManual), FromMode: string(model.CrawlModeNormal), ToMode: string(model.CrawlModeCrisis), FromIntervalMinutes: 10, ToIntervalMinutes: 10, Reason: "reason", EventRef: "event"}).Return(model.CrawlModeChange{}, nil)
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
		current model.CrawlTarget
		latest  model.DryrunResult
		update  model.CrawlTarget
	}

	tcs := map[string]struct {
		input  datasource.UpdateTargetInput
		mock   mockData
		output datasource.UpdateTargetOutput
		err    error
	}{
		"success": {
			input:  datasource.UpdateTargetInput{DataSourceID: testSourceID, ID: testTargetID, Values: []string{"new"}, CrawlIntervalMinutes: &interval},
			mock:   mockData{current: testCrawlTarget(true), update: testCrawlTarget(false)},
			output: datasource.UpdateTargetOutput{Target: testCrawlTarget(false)},
		},
		"invalid":    {err: datasource.ErrTargetNotFound},
		"repo_error": {input: datasource.UpdateTargetInput{DataSourceID: testSourceID, ID: testTargetID}, err: datasource.ErrTargetUpdateFailed},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			uc, r, _ := newDatasourceUC(t)
			if tc.input.ID != "" {
				if name == "repo_error" {
					r.EXPECT().GetTarget(context.Background(), repo.GetTargetOptions{DataSourceID: testSourceID, ID: testTargetID}).Return(model.CrawlTarget{}, errors.New("db"))
				} else {
					r.EXPECT().GetTarget(context.Background(), repo.GetTargetOptions{DataSourceID: testSourceID, ID: testTargetID}).Return(tc.mock.current, nil)
					r.EXPECT().GetLatestDryrunByTarget(context.Background(), testTargetID).Return(model.DryrunResult{}, nil)
					r.EXPECT().UpdateTarget(context.Background(), repo.UpdateTargetOptions{DataSourceID: testSourceID, ID: testTargetID, Values: model.TypesJSONFromStringSlice([]string{"new"}), IsActive: boolPtr(false), CrawlIntervalMinutes: &interval}).Return(tc.mock.update, nil)
					r.EXPECT().DetailDataSource(context.Background(), testSourceID).Return(testSource(model.SourceStatusReady), nil)
					r.EXPECT().UpdateDataSource(context.Background(), repo.UpdateDataSourceOptions{ID: testSourceID, Status: string(model.SourceStatusPending), DryrunStatus: string(model.DryrunStatusPending)}).Return(testSource(model.SourceStatusPending), nil)
				}
			}

			output, err := uc.UpdateTarget(context.Background(), tc.input)

			require.ErrorIs(t, err, tc.err)
			require.Equal(t, tc.output, output)
		})
	}
}

func TestActivateTarget(t *testing.T) {
	tcs := map[string]struct {
		input  datasource.ActivateTargetInput
		mock   model.CrawlTarget
		output datasource.ActivateTargetOutput
		err    error
	}{
		"success":        {input: datasource.ActivateTargetInput{DataSourceID: testSourceID, ID: testTargetID}, mock: testCrawlTarget(false), output: datasource.ActivateTargetOutput{Target: testCrawlTarget(true)}},
		"invalid":        {err: datasource.ErrTargetNotFound},
		"already_active": {input: datasource.ActivateTargetInput{DataSourceID: testSourceID, ID: testTargetID}, mock: testCrawlTarget(true), output: datasource.ActivateTargetOutput{Target: testCrawlTarget(true)}},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			uc, r, _ := newDatasourceUC(t)
			if tc.input.ID != "" {
				r.EXPECT().GetTarget(context.Background(), repo.GetTargetOptions{DataSourceID: testSourceID, ID: testTargetID}).Return(tc.mock, nil)
				r.EXPECT().DetailDataSource(context.Background(), testSourceID).Return(testSource(model.SourceStatusReady), nil)
				r.EXPECT().GetLatestDryrunByTarget(context.Background(), testTargetID).Return(model.DryrunResult{ID: "dryrun-1", Status: model.DryrunStatusSuccess}, nil)
				if name == "success" {
					r.EXPECT().GetLatestDryrunByTarget(context.Background(), testTargetID).Return(model.DryrunResult{ID: "dryrun-1", Status: model.DryrunStatusSuccess}, nil)
					r.EXPECT().UpdateTarget(context.Background(), repo.UpdateTargetOptions{DataSourceID: testSourceID, ID: testTargetID, IsActive: boolPtr(true)}).Return(testCrawlTarget(true), nil)
				}
			}

			output, err := uc.ActivateTarget(context.Background(), tc.input)

			require.ErrorIs(t, err, tc.err)
			require.Equal(t, tc.output, output)
		})
	}
}

func TestDeactivateTarget(t *testing.T) {
	tcs := map[string]struct {
		input  datasource.DeactivateTargetInput
		mock   model.CrawlTarget
		output datasource.DeactivateTargetOutput
		err    error
	}{
		"success":          {input: datasource.DeactivateTargetInput{DataSourceID: testSourceID, ID: testTargetID}, mock: testCrawlTarget(true), output: datasource.DeactivateTargetOutput{Target: testCrawlTarget(false)}},
		"invalid":          {err: datasource.ErrTargetNotFound},
		"already_inactive": {input: datasource.DeactivateTargetInput{DataSourceID: testSourceID, ID: testTargetID}, mock: testCrawlTarget(false), output: datasource.DeactivateTargetOutput{Target: testCrawlTarget(false)}},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			uc, r, _ := newDatasourceUC(t)
			if tc.input.ID != "" {
				r.EXPECT().GetTarget(context.Background(), repo.GetTargetOptions{DataSourceID: testSourceID, ID: testTargetID}).Return(tc.mock, nil)
				r.EXPECT().GetLatestDryrunByTarget(context.Background(), testTargetID).Return(model.DryrunResult{}, nil)
				if name == "success" {
					r.EXPECT().DetailDataSource(context.Background(), testSourceID).Return(testSource(model.SourceStatusActive), nil)
					r.EXPECT().CountActiveTargets(context.Background(), testSourceID).Return(int64(2), nil)
					r.EXPECT().UpdateTarget(context.Background(), repo.UpdateTargetOptions{DataSourceID: testSourceID, ID: testTargetID, IsActive: boolPtr(false)}).Return(testCrawlTarget(false), nil)
				}
			}

			output, err := uc.DeactivateTarget(context.Background(), tc.input)

			require.ErrorIs(t, err, tc.err)
			require.Equal(t, tc.output, output)
		})
	}
}

func TestDeleteTarget(t *testing.T) {
	tcs := map[string]struct {
		input  datasource.DeleteTargetInput
		mock   model.CrawlTarget
		output struct{}
		err    error
	}{
		"success":    {input: datasource.DeleteTargetInput{DataSourceID: testSourceID, ID: testTargetID}, mock: testCrawlTarget(false)},
		"invalid":    {err: datasource.ErrTargetNotFound},
		"repo_error": {input: datasource.DeleteTargetInput{DataSourceID: testSourceID, ID: testTargetID}, err: datasource.ErrTargetDeleteFailed},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			uc, r, _ := newDatasourceUC(t)
			if tc.input.ID != "" {
				if name == "repo_error" {
					r.EXPECT().GetTarget(context.Background(), repo.GetTargetOptions{DataSourceID: testSourceID, ID: testTargetID}).Return(model.CrawlTarget{}, errors.New("db"))
				} else {
					r.EXPECT().GetTarget(context.Background(), repo.GetTargetOptions{DataSourceID: testSourceID, ID: testTargetID}).Return(tc.mock, nil)
					r.EXPECT().GetLatestDryrunByTarget(context.Background(), testTargetID).Return(model.DryrunResult{}, nil)
					r.EXPECT().DeleteTarget(context.Background(), repo.DeleteTargetOptions{DataSourceID: testSourceID, ID: testTargetID}).Return(nil)
				}
			}

			err := uc.DeleteTarget(context.Background(), tc.input)

			require.ErrorIs(t, err, tc.err)
		})
	}
}

func TestGetActivationReadiness(t *testing.T) {
	tcs := map[string]struct {
		input  datasource.ActivationReadinessInput
		mock   []model.DataSource
		output bool
		err    error
	}{
		"success": {input: datasource.ActivationReadinessInput{ProjectID: testProjectID}, mock: []model.DataSource{testSource(model.SourceStatusReady)}, output: true},
		"invalid": {input: datasource.ActivationReadinessInput{}, err: datasource.ErrProjectIDRequired},
		"empty":   {input: datasource.ActivationReadinessInput{ProjectID: testProjectID}, output: false},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			uc, r, _ := newDatasourceUC(t)
			if tc.input.ProjectID != "" {
				r.EXPECT().ListDataSources(context.Background(), repo.ListDataSourcesOptions{ProjectID: testProjectID}).Return(tc.mock, nil)
				if name == "success" {
					r.EXPECT().ListTargets(context.Background(), repo.ListTargetsOptions{DataSourceID: testSourceID}).Return([]model.CrawlTarget{testCrawlTarget(true)}, nil)
					r.EXPECT().GetLatestDryrunByTarget(context.Background(), testTargetID).Return(model.DryrunResult{ID: "dryrun-1", Status: model.DryrunStatusSuccess}, nil)
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
	tcs := map[string]struct {
		input  string
		mock   struct{}
		output datasource.ProjectLifecycleOutput
		err    error
	}{
		"success": {input: testProjectID, output: datasource.ProjectLifecycleOutput{ProjectID: testProjectID, AffectedDataSourceCount: 1}},
		"invalid": {err: datasource.ErrProjectIDRequired},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			uc, r, _ := newDatasourceUC(t)
			if tc.input != "" {
				r.EXPECT().ListDataSources(context.Background(), repo.ListDataSourcesOptions{ProjectID: testProjectID}).Return([]model.DataSource{testSource(model.SourceStatusReady)}, nil).Once()
				r.EXPECT().ListTargets(context.Background(), repo.ListTargetsOptions{DataSourceID: testSourceID}).Return([]model.CrawlTarget{testCrawlTarget(true)}, nil)
				r.EXPECT().GetLatestDryrunByTarget(context.Background(), testTargetID).Return(model.DryrunResult{ID: "dryrun-1", Status: model.DryrunStatusSuccess}, nil)
				r.EXPECT().ListDataSources(context.Background(), repo.ListDataSourcesOptions{ProjectID: testProjectID}).Return([]model.DataSource{testSource(model.SourceStatusReady)}, nil).Once()
				r.EXPECT().UpdateProjectDataSourcesLifecycle(context.Background(), uc.buildProjectLifecycleUpdateOptions(testProjectID, "activate")).Return(int64(1), nil)
			}

			output, err := uc.Activate(context.Background(), tc.input)

			require.ErrorIs(t, err, tc.err)
			require.Equal(t, tc.output, output)
		})
	}
}

func TestPause(t *testing.T) {
	tcs := map[string]struct {
		input  string
		mock   struct{}
		output datasource.ProjectLifecycleOutput
		err    error
	}{
		"success": {input: testProjectID, output: datasource.ProjectLifecycleOutput{ProjectID: testProjectID, AffectedDataSourceCount: 1}},
		"invalid": {err: datasource.ErrProjectIDRequired},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			uc, r, _ := newDatasourceUC(t)
			if tc.input != "" {
				r.EXPECT().UpdateProjectDataSourcesLifecycle(context.Background(), uc.buildProjectLifecycleUpdateOptions(testProjectID, "pause")).Return(int64(1), nil)
			}

			output, err := uc.Pause(context.Background(), tc.input)

			require.ErrorIs(t, err, tc.err)
			require.Equal(t, tc.output, output)
		})
	}
}

func TestResume(t *testing.T) {
	tcs := map[string]struct {
		input  string
		mock   struct{}
		output datasource.ProjectLifecycleOutput
		err    error
	}{
		"success": {input: testProjectID, output: datasource.ProjectLifecycleOutput{ProjectID: testProjectID, AffectedDataSourceCount: 1}},
		"invalid": {err: datasource.ErrProjectIDRequired},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			uc, r, _ := newDatasourceUC(t)
			if tc.input != "" {
				r.EXPECT().ListDataSources(context.Background(), repo.ListDataSourcesOptions{ProjectID: testProjectID}).Return([]model.DataSource{testSource(model.SourceStatusPaused)}, nil).Once()
				r.EXPECT().ListTargets(context.Background(), repo.ListTargetsOptions{DataSourceID: testSourceID}).Return([]model.CrawlTarget{testCrawlTarget(true)}, nil)
				r.EXPECT().GetLatestDryrunByTarget(context.Background(), testTargetID).Return(model.DryrunResult{ID: "dryrun-1", Status: model.DryrunStatusSuccess}, nil)
				r.EXPECT().ListDataSources(context.Background(), repo.ListDataSourcesOptions{ProjectID: testProjectID}).Return([]model.DataSource{testSource(model.SourceStatusPaused)}, nil).Once()
				r.EXPECT().UpdateProjectDataSourcesLifecycle(context.Background(), uc.buildProjectLifecycleUpdateOptions(testProjectID, "resume")).Return(int64(1), nil)
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
