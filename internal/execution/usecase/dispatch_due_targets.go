package usecase

import (
	"context"

	"ingest-srv/internal/execution"
	repo "ingest-srv/internal/execution/repository"
	"ingest-srv/internal/model"
)

func (uc *implUseCase) DispatchDueTargets(ctx context.Context, input execution.DispatchDueTargetsInput) (execution.DispatchDueTargetsOutput, error) {
	now := input.Now
	if now.IsZero() {
		now = uc.now()
	}

	limit := input.Limit
	if limit <= 0 {
		limit = 1
	}

	dueTargets, err := uc.repo.ListDueTargets(ctx, now, limit)
	if err != nil {
		uc.l.Errorf(ctx, "execution.usecase.DispatchDueTargets.ListDueTargets: %v", err)
		return execution.DispatchDueTargetsOutput{}, execution.ErrDispatchFailed
	}

	output := execution.DispatchDueTargetsOutput{
		DueCount: len(dueTargets),
	}

	for _, dueTarget := range dueTargets {
		effectiveInterval, intervalErr := computeEffectiveInterval(dueTarget.Source, dueTarget.Target)
		if intervalErr != nil {
			output.FailedCount++
			uc.l.Errorf(
				ctx,
				"execution.usecase.DispatchDueTargets.computeEffectiveInterval: source_id=%s target_id=%s err=%v",
				dueTarget.Source.ID,
				dueTarget.Target.ID,
				intervalErr,
			)
			continue
		}

		claimed, claimErr := uc.repo.ClaimTarget(ctx, repo.ClaimTargetOptions{
			SourceID:    dueTarget.Source.ID,
			TargetID:    dueTarget.Target.ID,
			ClaimedAt:   now,
			NextCrawlAt: now.Add(effectiveInterval),
		})
		if claimErr != nil {
			output.FailedCount++
			uc.l.Errorf(
				ctx,
				"execution.usecase.DispatchDueTargets.ClaimTarget: source_id=%s target_id=%s err=%v",
				dueTarget.Source.ID,
				dueTarget.Target.ID,
				claimErr,
			)
			continue
		}
		if !claimed {
			output.SkippedRaceCount++
			uc.l.Infof(
				ctx,
				"execution.usecase.DispatchDueTargets.claimSkipped: source_id=%s target_id=%s",
				dueTarget.Source.ID,
				dueTarget.Target.ID,
			)
			continue
		}

		output.ClaimedCount++

		if _, dispatchErr := uc.DispatchTarget(ctx, execution.DispatchTargetInput{
			DataSourceID: dueTarget.Source.ID,
			TargetID:     dueTarget.Target.ID,
			TriggerType:  model.TriggerTypeScheduled,
			ScheduledFor: now,
			RequestedAt:  now,
			CronExpr:     input.CronExpr,
		}); dispatchErr != nil {
			output.FailedCount++
			uc.l.Errorf(
				ctx,
				"execution.usecase.DispatchDueTargets.DispatchTarget: source_id=%s target_id=%s err=%v",
				dueTarget.Source.ID,
				dueTarget.Target.ID,
				dispatchErr,
			)
			continue
		}

		output.DispatchedCount++
	}

	return output, nil
}
