package postgre

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"ingest-srv/internal/datasource/repository"
	"ingest-srv/internal/model"
	"ingest-srv/internal/sqlboiler"

	"github.com/aarondl/null/v8"
	"github.com/aarondl/sqlboiler/v4/boil"
	"github.com/google/uuid"
)

// CreateCrawlModeChange inserts one crawl mode audit record.
func (r *implRepository) CreateCrawlModeChange(ctx context.Context, opt repository.CreateCrawlModeChangeOptions) (model.CrawlModeChange, error) {
	row := &sqlboiler.CrawlModeChange{
		ID:                  uuid.NewString(),
		SourceID:            opt.SourceID,
		ProjectID:           opt.ProjectID,
		TriggerType:         sqlboiler.TriggerType(opt.TriggerType),
		FromMode:            sqlboiler.CrawlMode(opt.FromMode),
		ToMode:              sqlboiler.CrawlMode(opt.ToMode),
		FromIntervalMinutes: opt.FromIntervalMinutes,
		ToIntervalMinutes:   opt.ToIntervalMinutes,
		TriggeredAt:         time.Now(),
	}

	if opt.Reason != "" {
		row.Reason = null.StringFrom(opt.Reason)
	}
	if opt.EventRef != "" {
		row.EventRef = null.StringFrom(opt.EventRef)
	}
	if opt.TriggeredBy != "" {
		row.TriggeredBy = null.StringFrom(opt.TriggeredBy)
	}

	if err := row.Insert(ctx, r.db, boil.Infer()); err != nil {
		r.l.Errorf(ctx, "datasource.repository.CreateCrawlModeChange.Insert: %v", err)
		return model.CrawlModeChange{}, repository.ErrCrawlModeChangeFailedToInsert
	}

	return *model.NewCrawlModeChangeFromDB(row), nil
}

// BulkApplyProjectCrawlMode flips crawl_mode for every eligible crawl
// datasource of a project in one transaction. Replaces the previous
// N-source × 3-roundtrip loop with two statements:
//
//  1. UPDATE ingest.data_sources SET crawl_mode = :mode … RETURNING id, crawl_mode_old
//  2. INSERT INTO ingest.crawl_mode_changes … VALUES (multi-row)
//
// Eligibility matches the prior usecase: CRAWL category, READY/ACTIVE/PAUSED
// status, non-null crawl_mode and positive crawl_interval_minutes. Rows whose
// crawl_mode already equals the target are counted as `AlreadyTarget` and
// skipped (no UPDATE, no audit row). Returns counters so the caller can build
// the same noopReason it used to.
func (r *implRepository) BulkApplyProjectCrawlMode(
	ctx context.Context, opt repository.BulkApplyProjectCrawlModeOptions,
) (repository.BulkApplyProjectCrawlModeOutput, error) {
	out := repository.BulkApplyProjectCrawlModeOutput{}
	projectID := strings.TrimSpace(opt.ProjectID)
	mode := strings.TrimSpace(opt.TargetMode)
	if projectID == "" || mode == "" {
		return out, repository.ErrFailedToUpdate
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		r.l.Errorf(ctx, "datasource.repository.BulkApplyProjectCrawlMode.BeginTx: %v", err)
		return out, repository.ErrFailedToUpdate
	}
	defer func() { _ = tx.Rollback() }()

	const countQuery = `
		SELECT
		  COUNT(*) FILTER (
		    WHERE source_category = 'CRAWL'
		      AND status IN ('READY','ACTIVE','PAUSED')
		      AND crawl_mode IS NOT NULL
		      AND crawl_interval_minutes IS NOT NULL
		      AND crawl_interval_minutes > 0
		  ) AS eligible,
		  COUNT(*) FILTER (
		    WHERE source_category = 'CRAWL'
		      AND status IN ('READY','ACTIVE','PAUSED')
		      AND crawl_mode IS NOT NULL
		      AND crawl_interval_minutes IS NOT NULL
		      AND crawl_interval_minutes > 0
		      AND crawl_mode::text = $2
		  ) AS already_target
		FROM ingest.data_sources
		WHERE project_id = $1 AND deleted_at IS NULL
	`
	var eligible, alreadyTarget int
	if err := tx.QueryRowContext(ctx, countQuery, projectID, mode).Scan(&eligible, &alreadyTarget); err != nil {
		r.l.Errorf(ctx, "datasource.repository.BulkApplyProjectCrawlMode.count: %v", err)
		return out, repository.ErrFailedToUpdate
	}
	out.Eligible = eligible
	out.AlreadyTarget = alreadyTarget

	if eligible == 0 || alreadyTarget == eligible {
		if err := tx.Commit(); err != nil {
			return out, repository.ErrFailedToUpdate
		}
		return out, nil
	}

	const updateQuery = `
		WITH targets AS (
		  SELECT id, crawl_mode::text AS from_mode, crawl_interval_minutes
		    FROM ingest.data_sources
		   WHERE project_id = $1
		     AND deleted_at IS NULL
		     AND source_category = 'CRAWL'
		     AND status IN ('READY','ACTIVE','PAUSED')
		     AND crawl_mode IS NOT NULL
		     AND crawl_interval_minutes IS NOT NULL
		     AND crawl_interval_minutes > 0
		     AND crawl_mode::text <> $2
		   FOR UPDATE SKIP LOCKED
		), updated AS (
		  UPDATE ingest.data_sources d
		     SET crawl_mode = $2::ingest.crawl_mode, updated_at = NOW()
		    FROM targets t
		   WHERE d.id = t.id
		  RETURNING t.id, t.from_mode, t.crawl_interval_minutes
		)
		SELECT id, from_mode, crawl_interval_minutes FROM updated
	`
	rows, err := tx.QueryContext(ctx, updateQuery, projectID, mode)
	if err != nil {
		r.l.Errorf(ctx, "datasource.repository.BulkApplyProjectCrawlMode.update: %v", err)
		return out, repository.ErrFailedToUpdate
	}

	type changed struct {
		sourceID        string
		fromMode        string
		intervalMinutes int
	}
	var updated []changed
	for rows.Next() {
		var c changed
		if err := rows.Scan(&c.sourceID, &c.fromMode, &c.intervalMinutes); err != nil {
			rows.Close()
			r.l.Errorf(ctx, "datasource.repository.BulkApplyProjectCrawlMode.scan: %v", err)
			return out, repository.ErrFailedToUpdate
		}
		updated = append(updated, c)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		r.l.Errorf(ctx, "datasource.repository.BulkApplyProjectCrawlMode.rowsErr: %v", err)
		return out, repository.ErrFailedToUpdate
	}
	rows.Close()

	out.Affected = len(updated)
	if out.Affected == 0 {
		if err := tx.Commit(); err != nil {
			return out, repository.ErrFailedToUpdate
		}
		return out, nil
	}

	// Build a multi-row INSERT in one roundtrip.
	const cols = 11
	now := time.Now()
	args := make([]interface{}, 0, cols*len(updated))
	values := make([]string, 0, len(updated))
	for i, c := range updated {
		base := i * cols
		values = append(values,
			"($"+itoa(base+1)+
				",$"+itoa(base+2)+
				",$"+itoa(base+3)+
				",$"+itoa(base+4)+
				",$"+itoa(base+5)+
				",$"+itoa(base+6)+
				",$"+itoa(base+7)+
				",$"+itoa(base+8)+
				",$"+itoa(base+9)+
				",$"+itoa(base+10)+
				",$"+itoa(base+11)+")")
		args = append(args,
			uuid.NewString(),
			c.sourceID,
			projectID,
			opt.TriggerType,
			c.fromMode,
			mode,
			c.intervalMinutes,
			c.intervalMinutes,
			nullableString(opt.Reason),
			nullableString(opt.EventRef),
			now,
		)
	}
	insertQuery := `
		INSERT INTO ingest.crawl_mode_changes (
			id, source_id, project_id, trigger_type,
			from_mode, to_mode, from_interval_minutes, to_interval_minutes,
			reason, event_ref, triggered_at
		) VALUES ` + strings.Join(values, ",")
	if _, err := tx.ExecContext(ctx, insertQuery, args...); err != nil {
		r.l.Errorf(ctx, "datasource.repository.BulkApplyProjectCrawlMode.insertChanges: %v", err)
		return out, repository.ErrCrawlModeChangeFailedToInsert
	}

	if err := tx.Commit(); err != nil {
		r.l.Errorf(ctx, "datasource.repository.BulkApplyProjectCrawlMode.Commit: %v", err)
		return out, repository.ErrFailedToUpdate
	}
	return out, nil
}

func nullableString(s string) interface{} {
	if strings.TrimSpace(s) == "" {
		return sql.NullString{}
	}
	return s
}

// itoa avoids an strconv import here; it's only called with small positive ints
// (column offsets) so a tiny digit-table version is fine.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [10]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
