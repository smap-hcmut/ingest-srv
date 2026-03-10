package postgre

import (
	"context"
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
