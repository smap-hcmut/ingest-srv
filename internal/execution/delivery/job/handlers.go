package job

import (
	"context"
	"time"

	"ingest-srv/internal/execution"
)

func (h handler) DispatchDueTargets() {
	ctx := context.Background()

	h.l.Debugf(ctx, "execution.delivery.job.DispatchDueTargets: Start scheduled dispatch")

	now := time.Now()
	if h.cfg.Timezone != "" {
		if loc, err := time.LoadLocation(h.cfg.Timezone); err == nil {
			now = now.In(loc)
		}
	}

	output, err := h.uc.DispatchDueTargets(ctx, execution.DispatchDueTargetsInput{
		Now:      now,
		Limit:    h.cfg.HeartbeatLimit,
		CronExpr: h.cfg.HeartbeatCron,
	})
	if err != nil {
		h.l.Errorf(ctx, "execution.delivery.job.DispatchDueTargets: %v", err)
		return
	}

	h.l.Debugf(
		ctx,
		"execution.delivery.job.DispatchDueTargets: End scheduled dispatch due_count=%d claimed_count=%d dispatched_count=%d skipped_race_count=%d failed_count=%d",
		output.DueCount,
		output.ClaimedCount,
		output.DispatchedCount,
		output.SkippedRaceCount,
		output.FailedCount,
	)
}
