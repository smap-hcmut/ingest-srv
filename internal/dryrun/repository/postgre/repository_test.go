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

	dryrunRepo "ingest-srv/internal/dryrun/repository"
	"ingest-srv/internal/model"
	"ingest-srv/internal/sqlboiler"

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
		output dryrunRepo.Repository
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

func TestCreateResult(t *testing.T) {
	errInsert := errors.New("insert")
	tcs := map[string]struct {
		input  dryrunRepo.CreateResultOptions
		mock   fakeDBConfig
		output model.DryrunResult
		err    error
	}{
		"success_full": {
			input: dryrunRepo.CreateResultOptions{SourceID: "source-1", ProjectID: "project-1", TargetID: "target-1", JobID: "job-1", Status: "running", RequestedBy: "user-1", SampleCount: 1},
		},
		"success_minimal": {
			input: dryrunRepo.CreateResultOptions{SourceID: "source-1", ProjectID: "project-1", Status: "running"},
		},
		"insert_error": {
			input: dryrunRepo.CreateResultOptions{SourceID: "source-1", ProjectID: "project-1", Status: "running"},
			mock:  fakeDBConfig{queryErr: errInsert, execErr: errInsert},
			err:   dryrunRepo.ErrFailedToInsert,
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			repo, closeDB := newFakeRepo(t, tc.mock)
			defer closeDB()
			output, err := repo.CreateResult(context.Background(), tc.input)
			require.ErrorIs(t, err, tc.err)
			if tc.err == nil {
				require.NotEmpty(t, output.ID)
			}
		})
	}
}

func TestGetByJobID(t *testing.T) {
	testDryrunRead(t, func(repo *implRepository) (model.DryrunResult, error) {
		return repo.GetByJobID(context.Background(), "job-1")
	})
}

func TestUpdateResult(t *testing.T) {
	errQuery := errors.New("query")
	errExec := errors.New("exec")
	completedAt := time.Date(2026, 5, 6, 10, 0, 0, 0, time.UTC)
	totalFound := 2

	tcs := map[string]struct {
		input  dryrunRepo.UpdateResultOptions
		mock   fakeDBConfig
		output model.DryrunResult
		err    error
	}{
		"success_full": {
			input: dryrunRepo.UpdateResultOptions{ID: "dryrun-1", Status: string(model.DryrunStatusSuccess), SampleCount: 1, CompletedAt: &completedAt, TotalFound: &totalFound, SampleData: []byte(`[]`), Warnings: []byte(`[]`), ErrorMessage: "err"},
		},
		"success_auto_completed_failed": {
			input: dryrunRepo.UpdateResultOptions{ID: "dryrun-1", Status: string(model.DryrunStatusFailed)},
		},
		"success_auto_completed_warning": {
			input: dryrunRepo.UpdateResultOptions{ID: "dryrun-1", Status: string(model.DryrunStatusWarning)},
		},
		"success_no_optional_fields": {
			input: dryrunRepo.UpdateResultOptions{ID: "dryrun-1", Status: string(model.DryrunStatusRunning)},
		},
		"not_found": {
			input: dryrunRepo.UpdateResultOptions{ID: "dryrun-1"},
			mock:  fakeDBConfig{emptyDryrun: true},
			err:   dryrunRepo.ErrNotFound,
		},
		"query_error": {
			input: dryrunRepo.UpdateResultOptions{ID: "dryrun-1"},
			mock:  fakeDBConfig{queryErr: errQuery},
			err:   dryrunRepo.ErrFailedToUpdate,
		},
		"update_error": {
			input: dryrunRepo.UpdateResultOptions{ID: "dryrun-1"},
			mock:  fakeDBConfig{execErr: errExec},
			err:   dryrunRepo.ErrFailedToUpdate,
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			repo, closeDB := newFakeRepo(t, tc.mock)
			defer closeDB()
			output, err := repo.UpdateResult(context.Background(), tc.input)
			require.ErrorIs(t, err, tc.err)
			if tc.err == nil {
				require.Equal(t, "dryrun-1", output.ID)
			}
		})
	}
}

func TestCompleteResult(t *testing.T) {
	errQuery := errors.New("query")
	errExec := errors.New("exec")
	errBegin := errors.New("begin")
	errCommit := errors.New("commit")
	completedAt := time.Date(2026, 5, 6, 10, 0, 0, 0, time.UTC)
	totalFound := 2

	tcs := map[string]struct {
		input  dryrunRepo.CompleteResultOptions
		mock   fakeDBConfig
		output model.DryrunResult
		err    error
	}{
		"success_activate_target": {
			input: dryrunRepo.CompleteResultOptions{ID: "dryrun-1", Status: string(model.DryrunStatusSuccess), SampleCount: 1, CompletedAt: &completedAt, TotalFound: &totalFound, SampleData: []byte(`[]`), Warnings: []byte(`[]`), ErrorMessage: "err", ActivateTarget: true},
			mock:  fakeDBConfig{sourceStatus: string(sqlboiler.SourceStatusPENDING), targetActive: false},
		},
		"success_failed_keeps_pending": {
			input: dryrunRepo.CompleteResultOptions{ID: "dryrun-1", Status: string(model.DryrunStatusFailed), ActivateTarget: false},
			mock:  fakeDBConfig{sourceStatus: string(sqlboiler.SourceStatusREADY)},
		},
		"success_no_target_found": {
			input: dryrunRepo.CompleteResultOptions{ID: "dryrun-1", Status: string(model.DryrunStatusSuccess), ActivateTarget: true},
			mock:  fakeDBConfig{emptyTarget: true},
		},
		"begin_error": {
			input: dryrunRepo.CompleteResultOptions{ID: "dryrun-1"},
			mock:  fakeDBConfig{beginErr: errBegin},
			err:   dryrunRepo.ErrFailedToUpdate,
		},
		"result_not_found": {
			input: dryrunRepo.CompleteResultOptions{ID: "dryrun-1"},
			mock:  fakeDBConfig{emptyDryrun: true},
			err:   dryrunRepo.ErrNotFound,
		},
		"result_query_error": {
			input: dryrunRepo.CompleteResultOptions{ID: "dryrun-1"},
			mock:  fakeDBConfig{queryErr: errQuery},
			err:   dryrunRepo.ErrFailedToUpdate,
		},
		"result_update_error": {
			input: dryrunRepo.CompleteResultOptions{ID: "dryrun-1"},
			mock:  fakeDBConfig{execErrAt: 1, execErr: errExec},
			err:   dryrunRepo.ErrFailedToUpdate,
		},
		"source_not_found": {
			input: dryrunRepo.CompleteResultOptions{ID: "dryrun-1"},
			mock:  fakeDBConfig{emptySource: true},
			err:   dryrunRepo.ErrNotFound,
		},
		"source_query_error": {
			input: dryrunRepo.CompleteResultOptions{ID: "dryrun-1"},
			mock:  fakeDBConfig{sourceQueryErr: errQuery},
			err:   dryrunRepo.ErrFailedToUpdate,
		},
		"source_deleted": {
			input: dryrunRepo.CompleteResultOptions{ID: "dryrun-1"},
			mock:  fakeDBConfig{deletedSource: true},
			err:   dryrunRepo.ErrNotFound,
		},
		"source_update_error": {
			input: dryrunRepo.CompleteResultOptions{ID: "dryrun-1"},
			mock:  fakeDBConfig{execErrAt: 2, execErr: errExec},
			err:   dryrunRepo.ErrFailedToUpdate,
		},
		"target_query_error": {
			input: dryrunRepo.CompleteResultOptions{ID: "dryrun-1", ActivateTarget: true},
			mock:  fakeDBConfig{targetQueryErr: errQuery},
			err:   dryrunRepo.ErrFailedToUpdate,
		},
		"target_update_error": {
			input: dryrunRepo.CompleteResultOptions{ID: "dryrun-1", ActivateTarget: true},
			mock:  fakeDBConfig{targetActive: false, execErrAt: 3, execErr: errExec},
			err:   dryrunRepo.ErrFailedToUpdate,
		},
		"commit_error": {
			input: dryrunRepo.CompleteResultOptions{ID: "dryrun-1"},
			mock:  fakeDBConfig{commitErr: errCommit},
			err:   dryrunRepo.ErrFailedToUpdate,
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			repo, closeDB := newFakeRepo(t, tc.mock)
			defer closeDB()
			output, source, err := repo.CompleteResult(context.Background(), tc.input)
			require.ErrorIs(t, err, tc.err)
			if tc.err == nil {
				require.Equal(t, "dryrun-1", output.ID)
				require.Equal(t, "source-1", source.ID)
			}
		})
	}
}

func TestGetLatest(t *testing.T) {
	testDryrunRead(t, func(repo *implRepository) (model.DryrunResult, error) {
		return repo.GetLatest(context.Background(), dryrunRepo.GetLatestOptions{SourceID: "source-1", TargetID: "target-1"})
	})
	t.Run("without_target", func(t *testing.T) {
		repo, closeDB := newFakeRepo(t, fakeDBConfig{})
		defer closeDB()
		output, err := repo.GetLatest(context.Background(), dryrunRepo.GetLatestOptions{SourceID: "source-1"})
		require.NoError(t, err)
		require.Equal(t, "dryrun-1", output.ID)
	})
}

func TestListHistory(t *testing.T) {
	errQuery := errors.New("query")
	tcs := map[string]struct {
		input  dryrunRepo.ListHistoryOptions
		mock   fakeDBConfig
		output int
		err    error
	}{
		"success_with_target":    {input: dryrunRepo.ListHistoryOptions{SourceID: "source-1", TargetID: "target-1", Paginator: paginator.PaginateQuery{Page: 1, Limit: 10}}, output: 1},
		"success_without_target": {input: dryrunRepo.ListHistoryOptions{SourceID: "source-1", Paginator: paginator.PaginateQuery{Page: 1, Limit: 10}}, output: 1},
		"count_error":            {input: dryrunRepo.ListHistoryOptions{SourceID: "source-1", Paginator: paginator.PaginateQuery{Page: 1, Limit: 10}}, mock: fakeDBConfig{countErr: errQuery}, err: dryrunRepo.ErrFailedToList},
		"query_error":            {input: dryrunRepo.ListHistoryOptions{SourceID: "source-1", Paginator: paginator.PaginateQuery{Page: 1, Limit: 10}}, mock: fakeDBConfig{selectErr: errQuery}, err: dryrunRepo.ErrFailedToList},
	}
	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			repo, closeDB := newFakeRepo(t, tc.mock)
			defer closeDB()
			output, pag, err := repo.ListHistory(context.Background(), tc.input)
			require.ErrorIs(t, err, tc.err)
			if tc.err == nil {
				require.Len(t, output, tc.output)
				require.Equal(t, int64(tc.output), pag.Count)
			}
		})
	}
}

func TestRollbackTx(t *testing.T) {
	tcs := map[string]struct {
		input  *sql.Tx
		mock   fakeDBConfig
		output struct{}
		err    error
	}{
		"nil": {},
		"tx":  {},
	}
	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			if name == "tx" {
				db := newFakeDB(t, fakeDBConfig{})
				defer db.Close()
				tx, err := db.Begin()
				require.NoError(t, err)
				tc.input = tx
			}
			require.NotPanics(t, func() { rollbackTx(tc.input) })
		})
	}
}

func testDryrunRead(t *testing.T, fn func(*implRepository) (model.DryrunResult, error)) {
	t.Helper()
	tcs := map[string]struct {
		input  struct{}
		mock   fakeDBConfig
		output model.DryrunResult
		err    error
	}{
		"success":     {},
		"not_found":   {mock: fakeDBConfig{emptyDryrun: true}, err: dryrunRepo.ErrNotFound},
		"query_error": {mock: fakeDBConfig{queryErr: errors.New("query")}, err: dryrunRepo.ErrFailedToGet},
	}
	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			repo, closeDB := newFakeRepo(t, tc.mock)
			defer closeDB()
			output, err := fn(repo)
			require.ErrorIs(t, err, tc.err)
			if tc.err == nil {
				require.Equal(t, "dryrun-1", output.ID)
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
	queryErr       error
	selectErr      error
	countErr       error
	targetQueryErr error
	sourceQueryErr error
	execErr        error
	execErrAt      int64
	beginErr       error
	commitErr      error
	emptyDryrun    bool
	emptySource    bool
	emptyTarget    bool
	deletedSource  bool
	sourceStatus   string
	targetActive   bool
	execCount      atomic.Int64
}

var (
	fakeDriverOnce sync.Once
	fakeDBSeq      atomic.Int64
	fakeDBConfigs  sync.Map
)

func newFakeDB(t *testing.T, cfg fakeDBConfig) *sql.DB {
	t.Helper()
	fakeDriverOnce.Do(func() {
		sql.Register("dryrun-postgre-test", fakeSQLDriver{})
	})
	name := fmt.Sprintf("db-%d", fakeDBSeq.Add(1))
	fakeDBConfigs.Store(name, &cfg)
	t.Cleanup(func() { fakeDBConfigs.Delete(name) })
	db, err := sql.Open("dryrun-postgre-test", name)
	require.NoError(t, err)
	return db
}

type fakeSQLDriver struct{}

func (fakeSQLDriver) Open(name string) (driver.Conn, error) {
	value, _ := fakeDBConfigs.Load(name)
	cfg, _ := value.(*fakeDBConfig)
	return &fakeSQLConn{cfg: cfg}, nil
}

type fakeSQLConn struct{ cfg *fakeDBConfig }

func (c *fakeSQLConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("prepare unsupported")
}
func (c *fakeSQLConn) Close() error { return nil }
func (c *fakeSQLConn) Begin() (driver.Tx, error) {
	return c.BeginTx(context.Background(), driver.TxOptions{})
}
func (c *fakeSQLConn) CheckNamedValue(*driver.NamedValue) error { return nil }
func (c *fakeSQLConn) BeginTx(context.Context, driver.TxOptions) (driver.Tx, error) {
	if c.cfg.beginErr != nil {
		return nil, c.cfg.beginErr
	}
	return fakeSQLTx{cfg: c.cfg}, nil
}
func (c *fakeSQLConn) ExecContext(context.Context, string, []driver.NamedValue) (driver.Result, error) {
	if c.cfg.execErr != nil {
		count := c.cfg.execCount.Add(1)
		if c.cfg.execErrAt == 0 || c.cfg.execErrAt == count {
			return nil, c.cfg.execErr
		}
	}
	return fakeSQLResult(1), nil
}
func (c *fakeSQLConn) QueryContext(_ context.Context, query string, _ []driver.NamedValue) (driver.Rows, error) {
	lower := strings.ToLower(query)
	if strings.Contains(lower, "count(") {
		if c.cfg.countErr != nil {
			return nil, c.cfg.countErr
		}
		return &fakeSQLRows{columns: []string{"count"}, values: [][]driver.Value{{int64(1)}}}, nil
	}
	if c.cfg.targetQueryErr != nil && strings.Contains(lower, "crawl_targets") {
		return nil, c.cfg.targetQueryErr
	}
	if c.cfg.sourceQueryErr != nil && strings.Contains(lower, "data_sources") {
		return nil, c.cfg.sourceQueryErr
	}
	if c.cfg.selectErr != nil && strings.HasPrefix(strings.TrimSpace(lower), "select") {
		return nil, c.cfg.selectErr
	}
	if c.cfg.queryErr != nil {
		return nil, c.cfg.queryErr
	}
	columns := columnsForQuery(query)
	if (c.cfg.emptyDryrun && strings.Contains(lower, "dryrun_results")) ||
		(c.cfg.emptySource && strings.Contains(lower, "data_sources")) ||
		(c.cfg.emptyTarget && strings.Contains(lower, "crawl_targets")) {
		return &fakeSQLRows{columns: columns}, nil
	}
	values := make([]driver.Value, len(columns))
	for i, col := range columns {
		values[i] = valueForColumn(query, col, c.cfg)
	}
	return &fakeSQLRows{columns: columns, values: [][]driver.Value{values}}, nil
}

type fakeSQLTx struct{ cfg *fakeDBConfig }

func (tx fakeSQLTx) Commit() error { return tx.cfg.commitErr }
func (fakeSQLTx) Rollback() error  { return nil }

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

func columnsForQuery(query string) []string {
	if returning := returningColumns(query); len(returning) > 0 {
		return returning
	}
	lower := strings.ToLower(query)
	switch {
	case strings.Contains(lower, "data_sources"):
		return []string{"id", "project_id", "name", "description", "source_type", "source_category", "status", "config", "account_ref", "mapping_rules", "onboarding_status", "dryrun_status", "dryrun_last_result_id", "crawl_mode", "crawl_interval_minutes", "next_crawl_at", "last_crawl_at", "last_success_at", "last_error_at", "last_error_message", "webhook_id", "webhook_secret_encrypted", "created_by", "activated_at", "paused_at", "archived_at", "created_at", "updated_at", "deleted_at"}
	case strings.Contains(lower, "crawl_targets"):
		return []string{"id", "data_source_id", "target_type", "values", "label", "platform_meta", "is_active", "priority", "crawl_interval_minutes", "next_crawl_at", "last_crawl_at", "last_success_at", "last_error_at", "last_error_message", "created_at", "updated_at"}
	default:
		return []string{"id", "source_id", "project_id", "target_id", "job_id", "status", "sample_count", "total_found", "sample_data", "warnings", "error_message", "requested_by", "started_at", "completed_at", "created_at"}
	}
}

func returningColumns(query string) []string {
	idx := strings.Index(strings.ToLower(query), " returning ")
	if idx < 0 {
		return nil
	}
	part := strings.ReplaceAll(query[idx+len(" returning "):], `"`, "")
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

func valueForColumn(query, col string, cfg *fakeDBConfig) driver.Value {
	now := time.Date(2026, 5, 6, 10, 0, 0, 0, time.UTC)
	lower := strings.ToLower(query)
	switch col {
	case "id":
		if strings.Contains(lower, "data_sources") {
			return "source-1"
		}
		if strings.Contains(lower, "crawl_targets") {
			return "target-1"
		}
		return "dryrun-1"
	case "source_id", "data_source_id":
		return "source-1"
	case "project_id":
		return "project-1"
	case "target_id":
		return "target-1"
	case "job_id":
		return "job-1"
	case "status":
		if strings.Contains(lower, "data_sources") {
			if cfg.sourceStatus != "" {
				return cfg.sourceStatus
			}
			return string(sqlboiler.SourceStatusACTIVE)
		}
		return string(model.DryrunStatusSuccess)
	case "sample_count", "total_found", "priority", "crawl_interval_minutes":
		return int64(1)
	case "sample_data", "warnings", "config", "account_ref", "mapping_rules", "values", "platform_meta":
		return []byte(`{}`)
	case "error_message", "requested_by", "description", "name", "last_error_message", "webhook_id", "webhook_secret_encrypted", "created_by", "dryrun_last_result_id", "label":
		return "value"
	case "source_type":
		return "api"
	case "source_category":
		return "social"
	case "onboarding_status", "dryrun_status":
		return "not_required"
	case "crawl_mode":
		return "scheduled"
	case "target_type":
		return "keyword"
	case "is_active":
		return cfg.targetActive
	case "deleted_at":
		if cfg.deletedSource {
			return now
		}
		return nil
	case "next_crawl_at", "last_crawl_at", "last_success_at", "last_error_at", "activated_at", "paused_at", "archived_at":
		return nil
	case "started_at", "completed_at", "created_at", "updated_at":
		return now
	default:
		return nil
	}
}
