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

func (r *implRepository) ListDueTargets(ctx context.Context, now time.Time, limit int) ([]repo.DueTarget, error) {
	if limit <= 0 {
		limit = 1
	}

	rows, err := sqlboiler.CrawlTargets(
		qm.InnerJoin(fmt.Sprintf("%s ON %s.id = %s.data_source_id", sqlboiler.TableNames.DataSources, sqlboiler.TableNames.DataSources, sqlboiler.TableNames.CrawlTargets)),
		qm.Where(fmt.Sprintf("%s.status = ?", sqlboiler.TableNames.DataSources), model.SourceStatusActive),
		qm.Where(fmt.Sprintf("%s.source_category = ?", sqlboiler.TableNames.DataSources), model.SourceCategoryCrawl),
		qm.Where(fmt.Sprintf("%s.is_active = ?", sqlboiler.TableNames.CrawlTargets), true),
		qm.Where(fmt.Sprintf("(%s.next_crawl_at IS NULL OR %s.next_crawl_at <= ?)", sqlboiler.TableNames.CrawlTargets, sqlboiler.TableNames.CrawlTargets), now),
		qm.Load(sqlboiler.CrawlTargetRels.DataSource),
		qm.OrderBy(fmt.Sprintf("%s.next_crawl_at ASC NULLS FIRST", sqlboiler.TableNames.CrawlTargets)),
		qm.OrderBy(fmt.Sprintf("%s.priority DESC", sqlboiler.TableNames.CrawlTargets)),
		qm.OrderBy(fmt.Sprintf("%s.created_at ASC", sqlboiler.TableNames.CrawlTargets)),
		qm.Limit(limit),
	).All(ctx, r.db)
	if err != nil {
		r.l.Errorf(ctx, "execution.repository.ListDueTargets.Query: %v", err)
		return nil, repo.ErrListDueTargets
	}

	output := make([]repo.DueTarget, 0, len(rows))
	for _, row := range rows {
		if row == nil || row.R == nil || row.R.GetDataSource() == nil {
			continue
		}

		source := model.NewDataSourceFromDB(row.R.GetDataSource())
		target := model.NewCrawlTargetFromDB(row)
		if source == nil || target == nil {
			continue
		}

		output = append(output, repo.DueTarget{
			Source: *source,
			Target: *target,
		})
	}

	return output, nil
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
