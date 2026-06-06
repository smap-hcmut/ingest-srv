package postgre

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	repo "ingest-srv/internal/execution/repository"
	"ingest-srv/internal/model"
	"ingest-srv/internal/sqlboiler"

	"github.com/aarondl/null/v8"
	"github.com/aarondl/sqlboiler/v4/boil"
	"github.com/aarondl/sqlboiler/v4/queries/qm"
)

// ListDueTargets returns due targets interleaved across projects using a
// deficit-round-robin ranking (ROW_NUMBER OVER PARTITION BY project_id).
// Previously the query sorted globally on (next_crawl_at, priority) which let
// a single high-priority project monopolize every dispatch heartbeat — that
// produced the 12.5x p50/p95 TTFD gap documented in
// report/documents/docs/indexing-time-to-first-data-benchmark.md.
//
// Ranking ties inside one project keep the original sort (earliest
// next_crawl_at first, then higher priority, then earliest created_at) so
// per-project ordering does not regress.
func (r *implRepository) ListDueTargets(ctx context.Context, now time.Time, limit int) ([]repo.DueTarget, error) {
	if limit <= 0 {
		limit = 1
	}

	ids, err := r.rankDueTargetIDs(ctx, now, limit)
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return nil, nil
	}

	rows, err := sqlboiler.CrawlTargets(
		qm.WhereIn(fmt.Sprintf("%s.id IN ?", sqlboiler.TableNames.CrawlTargets), toAnySlice(ids)...),
		qm.Load(sqlboiler.CrawlTargetRels.DataSource),
	).All(ctx, r.db)
	if err != nil {
		r.l.Errorf(ctx, "execution.repository.ListDueTargets.Query: %v", err)
		return nil, repo.ErrListDueTargets
	}

	// Preserve DRR order from rankDueTargetIDs even though qm.WhereIn returns
	// rows in arbitrary order.
	byID := make(map[string]repo.DueTarget, len(rows))
	for _, row := range rows {
		if row == nil || row.R == nil || row.R.GetDataSource() == nil {
			continue
		}

		source := model.NewDataSourceFromDB(row.R.GetDataSource())
		target := model.NewCrawlTargetFromDB(row)
		if source == nil || target == nil {
			continue
		}

		byID[row.ID] = repo.DueTarget{
			Source: *source,
			Target: *target,
		}
	}

	output := make([]repo.DueTarget, 0, len(ids))
	for _, id := range ids {
		if dt, ok := byID[id]; ok {
			output = append(output, dt)
		}
	}

	return output, nil
}

// rankDueTargetIDs runs the round-robin ranking entirely in Postgres and
// returns the chosen IDs in dispatch order.
func (r *implRepository) rankDueTargetIDs(ctx context.Context, now time.Time, limit int) ([]string, error) {
	const query = `
WITH due AS (
    SELECT ct.id,
           ds.project_id,
           ROW_NUMBER() OVER (
               PARTITION BY ds.project_id
               ORDER BY ct.next_crawl_at ASC NULLS FIRST,
                        ct.priority DESC,
                        ct.created_at ASC
           ) AS rn
    FROM crawl_targets ct
    INNER JOIN data_sources ds ON ds.id = ct.data_source_id
    WHERE ds.status = $1
      AND ds.source_category = $2
      AND ct.is_active = TRUE
      AND (ct.next_crawl_at IS NULL OR ct.next_crawl_at <= $3)
)
SELECT id
FROM due
ORDER BY rn ASC, project_id ASC
LIMIT $4
`
	rows, err := r.db.QueryContext(ctx, query,
		model.SourceStatusActive,
		model.SourceCategoryCrawl,
		now,
		limit,
	)
	if err != nil {
		r.l.Errorf(ctx, "execution.repository.ListDueTargets.Rank: %v", err)
		return nil, repo.ErrListDueTargets
	}
	defer rows.Close()

	ids := make([]string, 0, limit)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			r.l.Errorf(ctx, "execution.repository.ListDueTargets.Scan: %v", err)
			return nil, repo.ErrListDueTargets
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		r.l.Errorf(ctx, "execution.repository.ListDueTargets.RowsErr: %v", err)
		return nil, repo.ErrListDueTargets
	}
	return ids, nil
}

func toAnySlice(s []string) []interface{} {
	out := make([]interface{}, len(s))
	for i, v := range s {
		out[i] = v
	}
	return out
}

func (r *implRepository) ClaimTarget(ctx context.Context, opt repo.ClaimTargetOptions) (bool, error) {
	query := fmt.Sprintf(`
UPDATE %s
SET next_crawl_at = $1,
    last_crawl_at = $2,
    updated_at = $2
WHERE id = $3
  AND data_source_id = $4
  AND is_active = TRUE
  AND (next_crawl_at IS NULL OR next_crawl_at <= $2)
  AND EXISTS (
    SELECT 1
    FROM %s
    WHERE %s.id = %s.data_source_id
      AND %s.status = $5
      AND %s.source_category = $6
  )
`, sqlboiler.TableNames.CrawlTargets, sqlboiler.TableNames.DataSources, sqlboiler.TableNames.DataSources, sqlboiler.TableNames.CrawlTargets, sqlboiler.TableNames.DataSources, sqlboiler.TableNames.DataSources)

	result, err := r.db.ExecContext(
		ctx,
		query,
		opt.NextCrawlAt,
		opt.ClaimedAt,
		opt.TargetID,
		opt.SourceID,
		model.SourceStatusActive,
		model.SourceCategoryCrawl,
	)
	if err != nil {
		r.l.Errorf(ctx, "execution.repository.ClaimTarget.Exec: %v", err)
		return false, repo.ErrClaimTarget
	}

	affected, err := result.RowsAffected()
	if err != nil {
		r.l.Errorf(ctx, "execution.repository.ClaimTarget.RowsAffected: %v", err)
		return false, repo.ErrClaimTarget
	}

	return affected > 0, nil
}

func (r *implRepository) ReleaseClaimTarget(ctx context.Context, opt repo.ReleaseClaimTargetOptions) error {
	targetID := strings.TrimSpace(opt.TargetID)
	sourceID := strings.TrimSpace(opt.SourceID)
	if targetID == "" || sourceID == "" {
		return nil
	}

	row, err := sqlboiler.CrawlTargets(
		sqlboiler.CrawlTargetWhere.ID.EQ(targetID),
		sqlboiler.CrawlTargetWhere.DataSourceID.EQ(sourceID),
	).One(ctx, r.db)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		r.l.Errorf(ctx, "execution.repository.ReleaseClaimTarget.FindTarget: %v", err)
		return repo.ErrClaimTarget
	}

	row.NextCrawlAt = null.Time{}
	row.LastCrawlAt = fallbackLastCrawlAt(row.LastSuccessAt, row.LastErrorAt)
	if _, err := row.Update(ctx, r.db, boil.Whitelist(
		sqlboiler.CrawlTargetColumns.NextCrawlAt,
		sqlboiler.CrawlTargetColumns.LastCrawlAt,
		sqlboiler.CrawlTargetColumns.UpdatedAt,
	)); err != nil {
		r.l.Errorf(ctx, "execution.repository.ReleaseClaimTarget.UpdateTarget: %v", err)
		return repo.ErrClaimTarget
	}

	return nil
}
