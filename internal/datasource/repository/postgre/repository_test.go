package postgre

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"ingest-srv/internal/datasource/repository"
	"ingest-srv/internal/model"
	"ingest-srv/internal/sqlboiler"

	"github.com/aarondl/sqlboiler/v4/types"
	"github.com/smap-hcmut/shared-libs/go/log"
	"github.com/smap-hcmut/shared-libs/go/paginator"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	db := newFakeDB(t, fakeDBConfig{})
	defer db.Close()

	tcs := map[string]struct {
		input  struct{}
		mock   struct{}
		output repository.Repository
		err    error
	}{
		"success": {},
	}

	for name := range tcs {
		t.Run(name, func(t *testing.T) {
			require.NotNil(t, New(testLogger(), db))
		})
	}
}

func TestCreateDataSource(t *testing.T) {
	errExec := errors.New("insert")

	tcs := map[string]struct {
		input  repository.CreateDataSourceOptions
		mock   fakeDBConfig
		output model.DataSource
		err    error
	}{
		"success_full": {
			input: repository.CreateDataSourceOptions{
				ProjectID: "project-1", Name: "source", Description: "desc", SourceType: "api", SourceCategory: "social",
				Config: []byte(`{"a":1}`), AccountRef: []byte(`{"b":2}`), MappingRules: []byte(`{"c":3}`),
				CrawlMode: "scheduled", CrawlIntervalMinutes: 15, WebhookID: "webhook-1", WebhookSecretEncrypted: "secret", CreatedBy: "user-1",
			},
		},
		"success_minimal": {
			input: repository.CreateDataSourceOptions{ProjectID: "project-1", Name: "source", SourceType: "api", SourceCategory: "social"},
		},
		"insert_error": {
			input: repository.CreateDataSourceOptions{ProjectID: "project-1", Name: "source", SourceType: "api", SourceCategory: "social"},
			mock:  fakeDBConfig{queryErr: errExec, execErr: errExec},
			err:   repository.ErrFailedToInsert,
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			repo, closeDB := newFakeRepo(t, tc.mock)
			defer closeDB()

			output, err := repo.CreateDataSource(context.Background(), tc.input)

			require.ErrorIs(t, err, tc.err)
			if tc.err == nil {
				require.NotEmpty(t, output.ID)
			}
		})
	}
}

func TestDetailDataSource(t *testing.T) {
	errQuery := errors.New("query")

	tcs := map[string]struct {
		input  string
		mock   fakeDBConfig
		output model.DataSource
		err    error
	}{
		"success": {input: "source-1"},
		"not_found": {
			input: "source-1",
			mock:  fakeDBConfig{empty: true},
		},
		"query_error": {
			input: "source-1",
			mock:  fakeDBConfig{queryErr: errQuery},
			err:   repository.ErrFailedToGet,
		},
		"soft_deleted": {
			input: "source-1",
			mock:  fakeDBConfig{deletedDataSource: true},
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			repo, closeDB := newFakeRepo(t, tc.mock)
			defer closeDB()

			output, err := repo.DetailDataSource(context.Background(), tc.input)

			require.ErrorIs(t, err, tc.err)
			if tc.err == nil && !tc.mock.empty && !tc.mock.deletedDataSource {
				require.Equal(t, "source-1", output.ID)
			}
		})
	}
}

func TestGetOneDataSource(t *testing.T) {
	errQuery := errors.New("query")

	tcs := map[string]struct {
		input  repository.GetOneDataSourceOptions
		mock   fakeDBConfig
		output model.DataSource
		err    error
	}{
		"success_all_filters": {
			input: repository.GetOneDataSourceOptions{ID: "source-1", ProjectID: "project-1", WebhookID: "webhook-1", Name: "source"},
		},
		"success_minimal_filters": {},
		"not_found": {
			mock: fakeDBConfig{empty: true},
		},
		"query_error": {
			mock: fakeDBConfig{queryErr: errQuery},
			err:  repository.ErrFailedToGet,
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			repo, closeDB := newFakeRepo(t, tc.mock)
			defer closeDB()

			output, err := repo.GetOneDataSource(context.Background(), tc.input)

			require.ErrorIs(t, err, tc.err)
			if tc.err == nil && !tc.mock.empty {
				require.Equal(t, "source-1", output.ID)
			}
		})
	}
}

func TestGetDataSources(t *testing.T) {
	errQuery := errors.New("query")

	tcs := map[string]struct {
		input  repository.GetDataSourcesOptions
		mock   fakeDBConfig
		output int
		err    error
	}{
		"success_all_filters": {
			input: repository.GetDataSourcesOptions{
				ProjectID: "project-1", Status: "active", SourceType: "api", SourceCategory: "social", CrawlMode: "scheduled", Name: "source",
				Paginator: paginator.PaginateQuery{Page: 2, Limit: 10},
			},
			output: 1,
		},
		"count_error": {
			input: repository.GetDataSourcesOptions{Paginator: paginator.PaginateQuery{Page: 1, Limit: 10}},
			mock:  fakeDBConfig{countErr: errQuery},
			err:   repository.ErrFailedToList,
		},
		"query_error": {
			input: repository.GetDataSourcesOptions{Paginator: paginator.PaginateQuery{Page: 1, Limit: 10}},
			mock:  fakeDBConfig{selectErr: errQuery},
			err:   repository.ErrFailedToList,
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			repo, closeDB := newFakeRepo(t, tc.mock)
			defer closeDB()

			output, pag, err := repo.GetDataSources(context.Background(), tc.input)

			require.ErrorIs(t, err, tc.err)
			if tc.err == nil {
				require.Len(t, output, tc.output)
				require.Equal(t, int64(tc.output), pag.Count)
			}
		})
	}
}

func TestListDataSources(t *testing.T) {
	errQuery := errors.New("query")
	tcs := map[string]struct {
		input  repository.ListDataSourcesOptions
		mock   fakeDBConfig
		output int
		err    error
	}{
		"success_all_filters": {
			input:  repository.ListDataSourcesOptions{ProjectID: "project-1", Status: "active", SourceType: "api", SourceCategory: "social", CrawlMode: "scheduled", Limit: 5},
			output: 1,
		},
		"success_minimal_filters": {output: 1},
		"query_error": {
			mock: fakeDBConfig{queryErr: errQuery},
			err:  repository.ErrFailedToList,
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			repo, closeDB := newFakeRepo(t, tc.mock)
			defer closeDB()

			output, err := repo.ListDataSources(context.Background(), tc.input)

			require.ErrorIs(t, err, tc.err)
			if tc.err == nil {
				require.Len(t, output, tc.output)
			}
		})
	}
}

func TestGetLatestDryrunByTarget(t *testing.T) {
	errQuery := errors.New("query")
	tcs := map[string]struct {
		input  string
		mock   fakeDBConfig
		output model.DryrunResult
		err    error
	}{
		"success": {input: "target-1"},
		"not_found": {
			input: "target-1",
			mock:  fakeDBConfig{empty: true},
		},
		"query_error": {
			input: "target-1",
			mock:  fakeDBConfig{queryErr: errQuery},
			err:   repository.ErrFailedToGet,
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			repo, closeDB := newFakeRepo(t, tc.mock)
			defer closeDB()

			output, err := repo.GetLatestDryrunByTarget(context.Background(), tc.input)

			require.ErrorIs(t, err, tc.err)
			if tc.err == nil && !tc.mock.empty {
				require.Equal(t, "dryrun-1", output.ID)
			}
		})
	}
}

func TestUpdateDataSource(t *testing.T) {
	errQuery := errors.New("query")
	errExec := errors.New("exec")
	interval := 15

	tcs := map[string]struct {
		input  repository.UpdateDataSourceOptions
		mock   fakeDBConfig
		output model.DataSource
		err    error
	}{
		"success_all_fields_active": {
			input: repository.UpdateDataSourceOptions{
				ID: "source-1", Name: "source", Description: "desc", Status: string(sqlboiler.SourceStatusACTIVE),
				Config: []byte(`{"a":1}`), AccountRef: []byte(`{"b":2}`), MappingRules: []byte(`{"c":3}`),
				OnboardingStatus: "not_required", DryrunStatus: "success", DryrunLastResultID: "dryrun-1",
				CrawlMode: "scheduled", CrawlIntervalMinutes: &interval, WebhookID: "webhook-1", WebhookSecretEncrypted: "secret", ClearPausedAt: true,
			},
		},
		"success_paused":             {input: repository.UpdateDataSourceOptions{ID: "source-1", Status: string(sqlboiler.SourceStatusPAUSED)}},
		"success_archived":           {input: repository.UpdateDataSourceOptions{ID: "source-1", Status: string(sqlboiler.SourceStatusARCHIVED)}},
		"success_no_optional_fields": {input: repository.UpdateDataSourceOptions{ID: "source-1"}},
		"not_found": {
			input: repository.UpdateDataSourceOptions{ID: "source-1"},
			mock:  fakeDBConfig{empty: true},
		},
		"query_error": {
			input: repository.UpdateDataSourceOptions{ID: "source-1"},
			mock:  fakeDBConfig{queryErr: errQuery},
			err:   repository.ErrFailedToGet,
		},
		"update_error": {
			input: repository.UpdateDataSourceOptions{ID: "source-1"},
			mock:  fakeDBConfig{execErr: errExec},
			err:   repository.ErrFailedToUpdate,
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			repo, closeDB := newFakeRepo(t, tc.mock)
			defer closeDB()

			output, err := repo.UpdateDataSource(context.Background(), tc.input)

			require.ErrorIs(t, err, tc.err)
			if tc.err == nil && !tc.mock.empty {
				require.Equal(t, "source-1", output.ID)
			}
		})
	}
}

func TestArchiveDataSource(t *testing.T) {
	testDataSourceWriteByID(t, func(repo *implRepository) error {
		return repo.ArchiveDataSource(context.Background(), "source-1")
	}, repository.ErrFailedToGet, repository.ErrFailedToDelete)
}

func TestDeleteDataSource(t *testing.T) {
	testDataSourceWriteByID(t, func(repo *implRepository) error {
		return repo.DeleteDataSource(context.Background(), "source-1")
	}, repository.ErrFailedToGet, repository.ErrFailedToDelete)
}

func TestCountActiveTargets(t *testing.T) {
	errQuery := errors.New("query")
	tcs := map[string]struct {
		input  string
		mock   fakeDBConfig
		output int64
		err    error
	}{
		"success":     {input: "source-1", output: 1},
		"query_error": {input: "source-1", mock: fakeDBConfig{queryErr: errQuery}, err: repository.ErrFailedToGet},
	}
	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			repo, closeDB := newFakeRepo(t, tc.mock)
			defer closeDB()
			output, err := repo.CountActiveTargets(context.Background(), tc.input)
			require.ErrorIs(t, err, tc.err)
			require.Equal(t, tc.output, output)
		})
	}
}

func TestUpdateProjectDataSourcesLifecycle(t *testing.T) {
	errExec := errors.New("exec")
	tcs := map[string]struct {
		input  repository.ProjectLifecycleUpdateOptions
		mock   fakeDBConfig
		output int64
		err    error
	}{
		"success_activated": {
			input:  repository.ProjectLifecycleUpdateOptions{ProjectID: "project-1", FromStatuses: []model.SourceStatus{"pending"}, ToStatus: "active", SetActivatedAt: true},
			output: 1,
		},
		"success_paused": {
			input:  repository.ProjectLifecycleUpdateOptions{ProjectID: "project-1", FromStatuses: []model.SourceStatus{"active"}, ToStatus: "paused", SetPausedAt: true},
			output: 1,
		},
		"success_clear_paused": {
			input:  repository.ProjectLifecycleUpdateOptions{ProjectID: "project-1", FromStatuses: []model.SourceStatus{"paused"}, ToStatus: "active", ClearPausedAt: true},
			output: 1,
		},
		"update_error": {
			input: repository.ProjectLifecycleUpdateOptions{ProjectID: "project-1", FromStatuses: []model.SourceStatus{"active"}, ToStatus: "paused"},
			mock:  fakeDBConfig{execErr: errExec},
			err:   repository.ErrFailedToUpdate,
		},
	}
	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			repo, closeDB := newFakeRepo(t, tc.mock)
			defer closeDB()
			output, err := repo.UpdateProjectDataSourcesLifecycle(context.Background(), tc.input)
			require.ErrorIs(t, err, tc.err)
			require.Equal(t, tc.output, output)
		})
	}
}

func TestCreateCrawlModeChange(t *testing.T) {
	errExec := errors.New("insert")
	tcs := map[string]struct {
		input  repository.CreateCrawlModeChangeOptions
		mock   fakeDBConfig
		output model.CrawlModeChange
		err    error
	}{
		"success_full": {
			input: repository.CreateCrawlModeChangeOptions{
				SourceID: "source-1", ProjectID: "project-1", TriggerType: "manual", FromMode: "manual", ToMode: "scheduled",
				FromIntervalMinutes: 1, ToIntervalMinutes: 2, Reason: "reason", EventRef: "event-1", TriggeredBy: "user-1",
			},
		},
		"success_minimal": {input: repository.CreateCrawlModeChangeOptions{SourceID: "source-1", ProjectID: "project-1", TriggerType: "manual", FromMode: "manual", ToMode: "scheduled"}},
		"insert_error": {
			input: repository.CreateCrawlModeChangeOptions{SourceID: "source-1", ProjectID: "project-1", TriggerType: "manual", FromMode: "manual", ToMode: "scheduled"},
			mock:  fakeDBConfig{queryErr: errExec, execErr: errExec},
			err:   repository.ErrCrawlModeChangeFailedToInsert,
		},
	}
	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			repo, closeDB := newFakeRepo(t, tc.mock)
			defer closeDB()
			output, err := repo.CreateCrawlModeChange(context.Background(), tc.input)
			require.ErrorIs(t, err, tc.err)
			if tc.err == nil {
				require.NotEmpty(t, output.ID)
			}
		})
	}
}

func TestCreateTarget(t *testing.T) {
	errExec := errors.New("insert")
	tcs := map[string]struct {
		input  repository.CreateTargetOptions
		mock   fakeDBConfig
		output model.CrawlTarget
		err    error
	}{
		"success_full": {
			input: repository.CreateTargetOptions{DataSourceID: "source-1", TargetType: "keyword", Values: types.JSON(`["a"]`), Label: "label", PlatformMeta: []byte(`{"x":1}`), IsActive: true, Priority: 1, CrawlIntervalMinutes: 15},
		},
		"success_minimal": {
			input: repository.CreateTargetOptions{DataSourceID: "source-1", TargetType: "keyword", Values: types.JSON(`["a"]`)},
		},
		"insert_error": {
			input: repository.CreateTargetOptions{DataSourceID: "source-1", TargetType: "keyword", Values: types.JSON(`["a"]`)},
			mock:  fakeDBConfig{queryErr: errExec, execErr: errExec},
			err:   repository.ErrTargetFailedToInsert,
		},
	}
	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			repo, closeDB := newFakeRepo(t, tc.mock)
			defer closeDB()
			output, err := repo.CreateTarget(context.Background(), tc.input)
			require.ErrorIs(t, err, tc.err)
			if tc.err == nil {
				require.NotEmpty(t, output.ID)
			}
		})
	}
}

func TestGetTarget(t *testing.T) {
	testTargetRead(t, func(repo *implRepository) (model.CrawlTarget, error) {
		return repo.GetTarget(context.Background(), repository.GetTargetOptions{DataSourceID: "source-1", ID: "target-1"})
	}, repository.ErrTargetNotFound)
}

func TestListTargets(t *testing.T) {
	errQuery := errors.New("query")
	active := true
	tcs := map[string]struct {
		input  repository.ListTargetsOptions
		mock   fakeDBConfig
		output int
		err    error
	}{
		"success_all_filters":     {input: repository.ListTargetsOptions{DataSourceID: "source-1", TargetType: "keyword", IsActive: &active}, output: 1},
		"success_minimal_filters": {input: repository.ListTargetsOptions{DataSourceID: "source-1"}, output: 1},
		"query_error":             {input: repository.ListTargetsOptions{DataSourceID: "source-1"}, mock: fakeDBConfig{queryErr: errQuery}, err: repository.ErrTargetFailedToList},
	}
	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			repo, closeDB := newFakeRepo(t, tc.mock)
			defer closeDB()
			output, err := repo.ListTargets(context.Background(), tc.input)
			require.ErrorIs(t, err, tc.err)
			if tc.err == nil {
				require.Len(t, output, tc.output)
			}
		})
	}
}

func TestUpdateTarget(t *testing.T) {
	errQuery := errors.New("query")
	errExec := errors.New("exec")
	active := false
	priority := 9
	interval := 30
	tcs := map[string]struct {
		input  repository.UpdateTargetOptions
		mock   fakeDBConfig
		output model.CrawlTarget
		err    error
	}{
		"success_all_fields": {
			input: repository.UpdateTargetOptions{DataSourceID: "source-1", ID: "target-1", Values: types.JSON(`["b"]`), Label: "new", PlatformMeta: []byte(`{"y":2}`), IsActive: &active, Priority: &priority, CrawlIntervalMinutes: &interval},
		},
		"success_no_optional_fields": {
			input: repository.UpdateTargetOptions{DataSourceID: "source-1", ID: "target-1"},
		},
		"not_found": {
			input: repository.UpdateTargetOptions{DataSourceID: "source-1", ID: "target-1"},
			mock:  fakeDBConfig{empty: true},
			err:   repository.ErrTargetNotFound,
		},
		"query_error": {
			input: repository.UpdateTargetOptions{DataSourceID: "source-1", ID: "target-1"},
			mock:  fakeDBConfig{queryErr: errQuery},
			err:   repository.ErrTargetFailedToUpdate,
		},
		"update_error": {
			input: repository.UpdateTargetOptions{DataSourceID: "source-1", ID: "target-1"},
			mock:  fakeDBConfig{execErr: errExec},
			err:   repository.ErrTargetFailedToUpdate,
		},
	}
	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			repo, closeDB := newFakeRepo(t, tc.mock)
			defer closeDB()
			output, err := repo.UpdateTarget(context.Background(), tc.input)
			require.ErrorIs(t, err, tc.err)
			if tc.err == nil {
				require.Equal(t, "target-1", output.ID)
			}
		})
	}
}

func TestDeleteTarget(t *testing.T) {
	tcs := map[string]struct {
		input  repository.DeleteTargetOptions
		mock   fakeDBConfig
		output struct{}
		err    error
	}{
		"success":      {input: repository.DeleteTargetOptions{DataSourceID: "source-1", ID: "target-1"}},
		"not_found":    {input: repository.DeleteTargetOptions{DataSourceID: "source-1", ID: "target-1"}, mock: fakeDBConfig{empty: true}, err: repository.ErrTargetNotFound},
		"query_error":  {input: repository.DeleteTargetOptions{DataSourceID: "source-1", ID: "target-1"}, mock: fakeDBConfig{queryErr: errors.New("query")}, err: repository.ErrTargetFailedToDelete},
		"delete_error": {input: repository.DeleteTargetOptions{DataSourceID: "source-1", ID: "target-1"}, mock: fakeDBConfig{execErr: errors.New("exec")}, err: repository.ErrTargetFailedToDelete},
	}
	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			repo, closeDB := newFakeRepo(t, tc.mock)
			defer closeDB()
			err := repo.DeleteTarget(context.Background(), tc.input)
			require.ErrorIs(t, err, tc.err)
		})
	}
}

func testDataSourceWriteByID(t *testing.T, fn func(*implRepository) error, notFoundErr, writeErr error) {
	t.Helper()
	tcs := map[string]struct {
		input  struct{}
		mock   fakeDBConfig
		output struct{}
		err    error
	}{
		"success":      {},
		"not_found":    {mock: fakeDBConfig{empty: true}, err: notFoundErr},
		"query_error":  {mock: fakeDBConfig{queryErr: errors.New("query")}, err: writeErr},
		"update_error": {mock: fakeDBConfig{execErr: errors.New("exec")}, err: writeErr},
	}
	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			repo, closeDB := newFakeRepo(t, tc.mock)
			defer closeDB()
			require.ErrorIs(t, fn(repo), tc.err)
		})
	}
}

func testTargetRead(t *testing.T, fn func(*implRepository) (model.CrawlTarget, error), expectedErr error) {
	t.Helper()
	tcs := map[string]struct {
		input  struct{}
		mock   fakeDBConfig
		output model.CrawlTarget
		err    error
	}{
		"success":     {},
		"not_found":   {mock: fakeDBConfig{empty: true}, err: expectedErr},
		"query_error": {mock: fakeDBConfig{queryErr: errors.New("query")}, err: expectedErr},
	}
	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			repo, closeDB := newFakeRepo(t, tc.mock)
			defer closeDB()
			output, err := fn(repo)
			require.ErrorIs(t, err, tc.err)
			if tc.err == nil {
				require.Equal(t, "target-1", output.ID)
			}
		})
	}
}

func newFakeRepo(t *testing.T, cfg fakeDBConfig) (*implRepository, func()) {
	t.Helper()
	db := newFakeDB(t, cfg)
	return New(testLogger(), db).(*implRepository), func() { _ = db.Close() }
}

func testLogger() log.Logger {
	return log.NewZapLogger(log.ZapConfig{Level: log.LevelFatal, Mode: log.ModeProduction, Encoding: log.EncodingJSON})
}

type fakeDBConfig struct {
	queryErr          error
	selectErr         error
	countErr          error
	execErr           error
	empty             bool
	deletedDataSource bool
	rowsAffected      int64
}

var (
	fakeDriverOnce sync.Once
	fakeDBSeq      atomic.Int64
	fakeDBConfigs  sync.Map
)

func newFakeDB(t *testing.T, cfg fakeDBConfig) *sql.DB {
	t.Helper()
	fakeDriverOnce.Do(func() {
		sql.Register("datasource-postgre-test", fakeSQLDriver{})
	})
	name := fmt.Sprintf("db-%d", fakeDBSeq.Add(1))
	if cfg.rowsAffected == 0 {
		cfg.rowsAffected = 1
	}
	fakeDBConfigs.Store(name, cfg)
	t.Cleanup(func() { fakeDBConfigs.Delete(name) })
	db, err := sql.Open("datasource-postgre-test", name)
	require.NoError(t, err)
	return db
}

type fakeSQLDriver struct{}

func (fakeSQLDriver) Open(name string) (driver.Conn, error) {
	value, _ := fakeDBConfigs.Load(name)
	cfg, _ := value.(fakeDBConfig)
	return &fakeSQLConn{cfg: cfg}, nil
}

type fakeSQLConn struct {
	cfg fakeDBConfig
}

func (c *fakeSQLConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("prepare unsupported")
}
func (c *fakeSQLConn) Close() error                             { return nil }
func (c *fakeSQLConn) Begin() (driver.Tx, error)                { return fakeSQLTx{}, nil }
func (c *fakeSQLConn) CheckNamedValue(*driver.NamedValue) error { return nil }

func (c *fakeSQLConn) ExecContext(_ context.Context, _ string, _ []driver.NamedValue) (driver.Result, error) {
	if c.cfg.execErr != nil {
		return nil, c.cfg.execErr
	}
	return fakeSQLResult(c.cfg.rowsAffected), nil
}

func (c *fakeSQLConn) QueryContext(_ context.Context, query string, _ []driver.NamedValue) (driver.Rows, error) {
	lower := strings.ToLower(query)
	if strings.Contains(lower, "count(") {
		if c.cfg.countErr != nil {
			return nil, c.cfg.countErr
		}
		if c.cfg.queryErr != nil {
			return nil, c.cfg.queryErr
		}
		return &fakeSQLRows{columns: []string{"count"}, values: [][]driver.Value{{int64(1)}}}, nil
	}
	if c.cfg.selectErr != nil && strings.HasPrefix(strings.TrimSpace(lower), "select") {
		return nil, c.cfg.selectErr
	}
	if c.cfg.queryErr != nil {
		return nil, c.cfg.queryErr
	}
	if c.cfg.empty && strings.HasPrefix(strings.TrimSpace(lower), "select") {
		return &fakeSQLRows{columns: columnsForQuery(query), values: nil}, nil
	}
	return rowsForQuery(query, c.cfg), nil
}

type fakeSQLTx struct{}

func (fakeSQLTx) Commit() error   { return nil }
func (fakeSQLTx) Rollback() error { return nil }

type fakeSQLResult int64

func (fakeSQLResult) LastInsertId() (int64, error)   { return 0, nil }
func (r fakeSQLResult) RowsAffected() (int64, error) { return int64(r), nil }

type fakeSQLRows struct {
	columns []string
	values  [][]driver.Value
	index   int
}

func (r *fakeSQLRows) Columns() []string { return r.columns }
func (r *fakeSQLRows) Close() error      { return nil }
func (r *fakeSQLRows) Next(dest []driver.Value) error {
	if r.index >= len(r.values) {
		return io.EOF
	}
	copy(dest, r.values[r.index])
	r.index++
	return nil
}

func rowsForQuery(query string, cfg fakeDBConfig) driver.Rows {
	columns := columnsForQuery(query)
	values := make([]driver.Value, len(columns))
	for i, col := range columns {
		values[i] = valueForColumn(query, col, cfg)
	}
	return &fakeSQLRows{columns: columns, values: [][]driver.Value{values}}
}

func columnsForQuery(query string) []string {
	if returning := returningColumns(query); len(returning) > 0 {
		return returning
	}
	lower := strings.ToLower(query)
	switch {
	case strings.Contains(lower, "crawl_targets"):
		return []string{"id", "data_source_id", "target_type", "values", "label", "platform_meta", "is_active", "priority", "crawl_interval_minutes", "next_crawl_at", "last_crawl_at", "last_success_at", "last_error_at", "last_error_message", "created_at", "updated_at"}
	case strings.Contains(lower, "dryrun_results"):
		return []string{"id", "source_id", "project_id", "target_id", "job_id", "status", "sample_count", "total_found", "sample_data", "warnings", "error_message", "requested_by", "started_at", "completed_at", "created_at"}
	case strings.Contains(lower, "crawl_mode_changes"):
		return []string{"id", "source_id", "project_id", "trigger_type", "from_mode", "to_mode", "from_interval_minutes", "to_interval_minutes", "reason", "event_ref", "triggered_by", "triggered_at", "created_at"}
	default:
		return []string{"id", "project_id", "name", "description", "source_type", "source_category", "status", "config", "account_ref", "mapping_rules", "onboarding_status", "dryrun_status", "dryrun_last_result_id", "crawl_mode", "crawl_interval_minutes", "next_crawl_at", "last_crawl_at", "last_success_at", "last_error_at", "last_error_message", "webhook_id", "webhook_secret_encrypted", "created_by", "activated_at", "paused_at", "archived_at", "created_at", "updated_at", "deleted_at"}
	}
}

func returningColumns(query string) []string {
	idx := strings.Index(strings.ToLower(query), " returning ")
	if idx < 0 {
		return nil
	}
	part := query[idx+len(" returning "):]
	part = strings.ReplaceAll(part, `"`, "")
	pieces := strings.Split(part, ",")
	columns := make([]string, 0, len(pieces))
	for _, piece := range pieces {
		piece = strings.TrimSpace(piece)
		if piece != "" {
			columns = append(columns, piece)
		}
	}
	return columns
}

func valueForColumn(query, col string, cfg fakeDBConfig) driver.Value {
	now := time.Date(2026, 5, 6, 10, 0, 0, 0, time.UTC)
	lower := strings.ToLower(query)
	switch col {
	case "id":
		switch {
		case strings.Contains(lower, "crawl_targets"):
			return "target-1"
		case strings.Contains(lower, "dryrun_results"):
			return "dryrun-1"
		case strings.Contains(lower, "crawl_mode_changes"):
			return "change-1"
		default:
			return "source-1"
		}
	case "project_id":
		return "project-1"
	case "source_id", "data_source_id":
		return "source-1"
	case "name":
		return "source"
	case "description", "label", "last_error_message", "error_message", "webhook_id", "webhook_secret_encrypted", "created_by", "dryrun_last_result_id", "target_id", "job_id", "requested_by", "reason", "event_ref", "triggered_by":
		return "value"
	case "source_type":
		return "api"
	case "source_category":
		return "social"
	case "status":
		if strings.Contains(lower, "dryrun_results") {
			return "success"
		}
		return "active"
	case "config", "account_ref", "mapping_rules", "values", "platform_meta", "sample_data", "warnings":
		return []byte(`{}`)
	case "onboarding_status":
		return "not_required"
	case "dryrun_status":
		return "not_required"
	case "crawl_mode", "from_mode", "to_mode":
		return "scheduled"
	case "crawl_interval_minutes", "priority", "sample_count", "total_found", "from_interval_minutes", "to_interval_minutes":
		return int64(1)
	case "is_active":
		return true
	case "target_type":
		return "keyword"
	case "trigger_type":
		return "manual"
	case "deleted_at":
		if cfg.deletedDataSource {
			return now
		}
		return nil
	case "next_crawl_at", "last_crawl_at", "last_success_at", "last_error_at", "activated_at", "paused_at", "archived_at", "started_at", "completed_at":
		return nil
	case "created_at", "updated_at", "triggered_at":
		return now
	default:
		return nil
	}
}
