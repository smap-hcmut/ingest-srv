package postgre

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"testing"
	"time"

	"ingest-srv/internal/model"
	repo "ingest-srv/internal/uap/repository"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/smap-hcmut/shared-libs/go/log"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	tcs := map[string]struct {
		input  struct{}
		mock   struct{}
		output repo.Repository
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

func TestClaimRawBatchForParsing(t *testing.T) {
	errExec := errors.New("exec")

	tcs := map[string]struct {
		input  string
		mock   func(sqlmock.Sqlmock)
		output bool
		err    error
	}{
		"success": {
			input: " raw-batch-1 ",
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`(?is).*raw_batches.*`).
					WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			output: true,
		},
		"no_rows": {
			input: "raw-batch-1",
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`(?is).*raw_batches.*`).
					WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
		},
		"update_error": {
			input: "raw-batch-1",
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`(?is).*raw_batches.*`).
					WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
					WillReturnError(errExec)
			},
			err: repo.ErrClaimRawBatch,
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			db, mock := newSQLMock(t)
			defer db.Close()
			tc.mock(mock)

			output, err := New(testLogger(), db).ClaimRawBatchForParsing(context.Background(), tc.input)

			require.ErrorIs(t, err, tc.err)
			require.Equal(t, tc.output, output)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestMarkRawBatchDownloaded(t *testing.T) {
	errExec := errors.New("exec")

	tcs := map[string]struct {
		input  repo.MarkRawBatchDownloadedOptions
		mock   func(sqlmock.Sqlmock)
		output struct{}
		err    error
	}{
		"success": {
			input: repo.MarkRawBatchDownloadedOptions{RawBatchID: "raw-batch-1"},
			mock: func(mock sqlmock.Sqlmock) {
				expectFindRawBatch(mock, "raw-batch-1")
				expectUpdateRawBatch(mock, 2, nil)
			},
		},
		"not_found": {
			input: repo.MarkRawBatchDownloadedOptions{RawBatchID: "raw-batch-1"},
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`(?is).*raw_batches.*`).
					WithArgs("raw-batch-1").
					WillReturnError(sql.ErrNoRows)
			},
			err: repo.ErrRawBatchNotFound,
		},
		"update_error": {
			input: repo.MarkRawBatchDownloadedOptions{RawBatchID: "raw-batch-1"},
			mock: func(mock sqlmock.Sqlmock) {
				expectFindRawBatch(mock, "raw-batch-1")
				expectUpdateRawBatch(mock, 2, errExec)
			},
			err: repo.ErrUpdateRawBatch,
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			db, mock := newSQLMock(t)
			defer db.Close()
			tc.mock(mock)

			err := New(testLogger(), db).MarkRawBatchDownloaded(context.Background(), tc.input)

			require.ErrorIs(t, err, tc.err)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestMarkRawBatchParsed(t *testing.T) {
	errExec := errors.New("exec")
	parsedAt := time.Date(2026, 5, 6, 10, 0, 0, 0, time.UTC)

	tcs := map[string]struct {
		input  repo.MarkRawBatchParsedOptions
		mock   func(sqlmock.Sqlmock)
		output struct{}
		err    error
	}{
		"success_with_metadata": {
			input: repo.MarkRawBatchParsedOptions{RawBatchID: "raw-batch-1", ParsedAt: parsedAt, PublishRecordCount: 10, RawMetadata: []byte(`{"ok":true}`)},
			mock: func(mock sqlmock.Sqlmock) {
				expectFindRawBatch(mock, "raw-batch-1")
				expectUpdateRawBatch(mock, 8, nil)
			},
		},
		"success_without_metadata": {
			input: repo.MarkRawBatchParsedOptions{RawBatchID: "raw-batch-1", ParsedAt: parsedAt, PublishRecordCount: 10},
			mock: func(mock sqlmock.Sqlmock) {
				expectFindRawBatch(mock, "raw-batch-1")
				expectUpdateRawBatch(mock, 8, nil)
			},
		},
		"not_found": {
			input: repo.MarkRawBatchParsedOptions{RawBatchID: "raw-batch-1"},
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`(?is).*raw_batches.*`).
					WithArgs("raw-batch-1").
					WillReturnError(sql.ErrNoRows)
			},
			err: repo.ErrRawBatchNotFound,
		},
		"update_error": {
			input: repo.MarkRawBatchParsedOptions{RawBatchID: "raw-batch-1", ParsedAt: parsedAt},
			mock: func(mock sqlmock.Sqlmock) {
				expectFindRawBatch(mock, "raw-batch-1")
				expectUpdateRawBatch(mock, 8, errExec)
			},
			err: repo.ErrUpdateRawBatch,
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			db, mock := newSQLMock(t)
			defer db.Close()
			tc.mock(mock)

			err := New(testLogger(), db).MarkRawBatchParsed(context.Background(), tc.input)

			require.ErrorIs(t, err, tc.err)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestMarkRawBatchFailed(t *testing.T) {
	errExec := errors.New("exec")

	tcs := map[string]struct {
		input  repo.MarkRawBatchFailedOptions
		mock   func(sqlmock.Sqlmock)
		output struct{}
		err    error
	}{
		"success_with_trimmed_errors_and_metadata": {
			input: repo.MarkRawBatchFailedOptions{
				RawBatchID:   "raw-batch-1",
				ErrorMessage: " failed ",
				PublishError: " publish ",
				RawMetadata:  []byte(`{"failed":true}`),
			},
			mock: func(mock sqlmock.Sqlmock) {
				expectFindRawBatch(mock, "raw-batch-1")
				expectUpdateRawBatch(mock, 6, nil)
			},
		},
		"success_without_errors_or_metadata": {
			input: repo.MarkRawBatchFailedOptions{RawBatchID: "raw-batch-1"},
			mock: func(mock sqlmock.Sqlmock) {
				expectFindRawBatch(mock, "raw-batch-1")
				expectUpdateRawBatch(mock, 6, nil)
			},
		},
		"not_found": {
			input: repo.MarkRawBatchFailedOptions{RawBatchID: "raw-batch-1"},
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`(?is).*raw_batches.*`).
					WithArgs("raw-batch-1").
					WillReturnError(sql.ErrNoRows)
			},
			err: repo.ErrRawBatchNotFound,
		},
		"update_error": {
			input: repo.MarkRawBatchFailedOptions{RawBatchID: "raw-batch-1"},
			mock: func(mock sqlmock.Sqlmock) {
				expectFindRawBatch(mock, "raw-batch-1")
				expectUpdateRawBatch(mock, 6, errExec)
			},
			err: repo.ErrUpdateRawBatch,
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			db, mock := newSQLMock(t)
			defer db.Close()
			tc.mock(mock)

			err := New(testLogger(), db).MarkRawBatchFailed(context.Background(), tc.input)

			require.ErrorIs(t, err, tc.err)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func newSQLMock(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	t.Helper()

	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	return db, mock
}

func expectFindRawBatch(mock sqlmock.Sqlmock, id string) {
	mock.ExpectQuery(`(?is).*raw_batches.*`).
		WithArgs(id).
		WillReturnRows(rawBatchRows())
}

func expectUpdateRawBatch(mock sqlmock.Sqlmock, args int, err error) {
	anyArgs := make([]driver.Value, args)
	for i := range anyArgs {
		anyArgs[i] = sqlmock.AnyArg()
	}
	expect := mock.ExpectExec(`(?is).*raw_batches.*`).WithArgs(anyArgs...)
	if err != nil {
		expect.WillReturnError(err)
		return
	}
	expect.WillReturnResult(sqlmock.NewResult(0, 1))
}

func rawBatchRows() *sqlmock.Rows {
	now := time.Date(2026, 5, 6, 10, 0, 0, 0, time.UTC)
	return sqlmock.NewRows([]string{
		"id", "source_id", "project_id", "domain_type_code", "external_task_id",
		"batch_id", "status", "storage_bucket", "storage_path", "storage_url",
		"item_count", "size_bytes", "checksum", "received_at", "parsed_at",
		"publish_status", "publish_record_count", "first_event_id", "last_event_id",
		"uap_published_at", "error_message", "publish_error", "raw_metadata", "created_at",
	}).AddRow(
		"raw-batch-1", "source-1", "project-1", "social_media", nil,
		"batch-1", model.BatchStatusReceived, "bucket", "path", nil,
		nil, nil, nil, now, nil,
		model.PublishStatusPending, 0, nil, nil,
		nil, nil, nil, []byte(`{}`), now,
	)
}

func testLogger() log.Logger {
	return log.NewZapLogger(log.ZapConfig{Level: log.LevelFatal, Mode: log.ModeProduction, Encoding: log.EncodingJSON})
}
