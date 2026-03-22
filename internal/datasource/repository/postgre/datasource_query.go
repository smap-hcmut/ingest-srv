package postgre

import (
	"ingest-srv/internal/datasource/repository"
	"ingest-srv/internal/sqlboiler"

	"github.com/aarondl/null/v8"
	"github.com/aarondl/sqlboiler/v4/queries/qm"
)

// buildGetOneQuery builds query mods for fetching a single data source by filters.
// All non-empty fields are applied as AND conditions.
func (r *implRepository) buildGetOneQuery(opt repository.GetOneDataSourceOptions) []qm.QueryMod {
	var mods []qm.QueryMod

	// Always exclude soft-deleted
	mods = append(mods, sqlboiler.DataSourceWhere.DeletedAt.IsNull())

	if opt.ID != "" {
		mods = append(mods, sqlboiler.DataSourceWhere.ID.EQ(opt.ID))
	}
	if opt.ProjectID != "" {
		mods = append(mods, sqlboiler.DataSourceWhere.ProjectID.EQ(opt.ProjectID))
	}
	if opt.WebhookID != "" {
		mods = append(mods, sqlboiler.DataSourceWhere.WebhookID.EQ(null.StringFrom(opt.WebhookID)))
	}
	if opt.Name != "" {
		mods = append(mods, sqlboiler.DataSourceWhere.Name.EQ(opt.Name))
	}

	return mods
}

// buildGetQuery builds filter-only query mods for paginated listing.
// Pagination (limit/offset/order) is NOT included here — applied in caller.
func (r *implRepository) buildGetQuery(opt repository.GetDataSourcesOptions) []qm.QueryMod {
	var mods []qm.QueryMod

	// Always exclude soft-deleted
	mods = append(mods, sqlboiler.DataSourceWhere.DeletedAt.IsNull())

	if opt.ProjectID != "" {
		mods = append(mods, sqlboiler.DataSourceWhere.ProjectID.EQ(opt.ProjectID))
	}
	if opt.Status != "" {
		mods = append(mods, sqlboiler.DataSourceWhere.Status.EQ(sqlboiler.SourceStatus(opt.Status)))
	}
	if opt.SourceType != "" {
		mods = append(mods, sqlboiler.DataSourceWhere.SourceType.EQ(sqlboiler.SourceType(opt.SourceType)))
	}
	if opt.SourceCategory != "" {
		mods = append(mods, sqlboiler.DataSourceWhere.SourceCategory.EQ(sqlboiler.SourceCategory(opt.SourceCategory)))
	}
	if opt.CrawlMode != "" {
		mods = append(mods, sqlboiler.DataSourceWhere.CrawlMode.EQ(sqlboiler.NullCrawlModeFrom(sqlboiler.CrawlMode(opt.CrawlMode))))
	}
	if opt.Name != "" {
		mods = append(mods, qm.Where(sqlboiler.DataSourceColumns.Name+" ILIKE ?", "%"+opt.Name+"%"))
	}

	return mods
}

// buildListQuery builds query mods for non-paginated listing.
func (r *implRepository) buildListQuery(opt repository.ListDataSourcesOptions) []qm.QueryMod {
	var mods []qm.QueryMod

	// Always exclude soft-deleted
	mods = append(mods, sqlboiler.DataSourceWhere.DeletedAt.IsNull())

	if opt.ProjectID != "" {
		mods = append(mods, sqlboiler.DataSourceWhere.ProjectID.EQ(opt.ProjectID))
	}
	if opt.Status != "" {
		mods = append(mods, sqlboiler.DataSourceWhere.Status.EQ(sqlboiler.SourceStatus(opt.Status)))
	}
	if opt.SourceType != "" {
		mods = append(mods, sqlboiler.DataSourceWhere.SourceType.EQ(sqlboiler.SourceType(opt.SourceType)))
	}
	if opt.SourceCategory != "" {
		mods = append(mods, sqlboiler.DataSourceWhere.SourceCategory.EQ(sqlboiler.SourceCategory(opt.SourceCategory)))
	}
	if opt.CrawlMode != "" {
		mods = append(mods, sqlboiler.DataSourceWhere.CrawlMode.EQ(sqlboiler.NullCrawlModeFrom(sqlboiler.CrawlMode(opt.CrawlMode))))
	}
	if opt.Limit > 0 {
		mods = append(mods, qm.Limit(opt.Limit))
	}

	// Default ordering
	mods = append(mods, qm.OrderBy(sqlboiler.DataSourceColumns.CreatedAt+" DESC"))

	return mods
}

func (r *implRepository) buildProjectLifecycleUpdateQuery(opt repository.ProjectLifecycleUpdateOptions) []qm.QueryMod {
	var statuses []sqlboiler.SourceStatus

	for _, status := range opt.FromStatuses {
		statuses = append(statuses, sqlboiler.SourceStatus(status))
	}

	return []qm.QueryMod{
		sqlboiler.DataSourceWhere.ProjectID.EQ(opt.ProjectID),
		sqlboiler.DataSourceWhere.DeletedAt.IsNull(),
		sqlboiler.DataSourceWhere.Status.IN(statuses),
	}
}
