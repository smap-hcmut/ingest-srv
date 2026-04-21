package usecase

import (
	"context"
	"time"

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

	// chon nhung target den han va co priority cao nhat trong so target do de dispatch truoc, tranh tinh trang target co priority thap bi bo qua lau qua va khong duoc dispatch
	uc.l.Infof(
		ctx,
		"execution.usecase.DispatchDueTargets.selected: now=%s limit=%d due_count=%d",
		now.Format(time.RFC3339),
		limit,
		len(dueTargets),
	)

	output := execution.DispatchDueTargetsOutput{
		DueCount: len(dueTargets),
	}

	for _, dueTarget := range dueTargets {
		dispatchCtx := repo.DispatchContext{
			Source: dueTarget.Source,
			Target: dueTarget.Target,
		}

		if err := uc.validateScheduledDispatchContext(dispatchCtx); err != nil {
			output.FailedCount++
			uc.l.Errorf(
				ctx,
				"execution.usecase.DispatchDueTargets.validateScheduledDispatchContext: source_id=%s target_id=%s err=%v",
				dueTarget.Source.ID,
				dueTarget.Target.ID,
				err,
			)
			continue
		}

		specs, err := uc.buildDispatchSpecs(dueTarget.Source, dueTarget.Target)
		if err != nil {
			output.FailedCount++
			uc.l.Errorf(
				ctx,
				"execution.usecase.DispatchDueTargets.buildDispatchSpecs: source_id=%s target_id=%s err=%v",
				dueTarget.Source.ID,
				dueTarget.Target.ID,
				err,
			)
			continue
		}

		effectiveInterval, intervalErr := uc.computeEffectiveInterval(dueTarget.Source, dueTarget.Target)
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

		// uc.l.Infof(
		// 	ctx,
		// 	"execution.usecase.DispatchDueTargets.interval: source_id=%s target_id=%s effective_interval=%s claimed_at=%s next_crawl_at=%s",
		// 	dueTarget.Source.ID,
		// 	dueTarget.Target.ID,
		// 	effectiveInterval,
		// 	now.Format(time.RFC3339),
		// 	now.Add(effectiveInterval).Format(time.RFC3339),
		// )

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

		currentDispatchCtx, currentErr := uc.repo.GetDispatchContext(ctx, dueTarget.Source.ID, dueTarget.Target.ID)
		if currentErr != nil {
			output.SkippedRaceCount++
			uc.l.Errorf(
				ctx,
				"execution.usecase.DispatchDueTargets.GetDispatchContext.afterClaim: source_id=%s target_id=%s err=%v",
				dueTarget.Source.ID,
				dueTarget.Target.ID,
				currentErr,
			)
			_ = uc.repo.ReleaseClaimTarget(ctx, repo.ReleaseClaimTargetOptions{
				SourceID: dueTarget.Source.ID,
				TargetID: dueTarget.Target.ID,
			})
			continue
		}

		if err := uc.validateScheduledDispatchContext(currentDispatchCtx); err != nil {
			output.SkippedRaceCount++
			uc.l.Errorf(
				ctx,
				"execution.usecase.DispatchDueTargets.validateScheduledDispatchContext.afterClaim: source_id=%s target_id=%s err=%v",
				dueTarget.Source.ID,
				dueTarget.Target.ID,
				err,
			)
			_ = uc.repo.ReleaseClaimTarget(ctx, repo.ReleaseClaimTargetOptions{
				SourceID: dueTarget.Source.ID,
				TargetID: dueTarget.Target.ID,
			})
			continue
		}

		// uc.l.Infof(
		// 	ctx,
		// 	"execution.usecase.DispatchDueTargets.claimed: source_id=%s target_id=%s next_crawl_at=%s",
		// 	dueTarget.Source.ID,
		// 	dueTarget.Target.ID,
		// 	now.Add(effectiveInterval).Format(time.RFC3339),
		// )

		dispatchOutput, dispatchErr := uc.dispatchPrepared(ctx, currentDispatchCtx, specs, execution.DispatchTargetInput{
			DataSourceID: dueTarget.Source.ID,
			TargetID:     dueTarget.Target.ID,
			TriggerType:  model.TriggerTypeScheduled,
			ScheduledFor: now,
			RequestedAt:  now,
			CronExpr:     input.CronExpr,
		})
		if dispatchErr != nil {
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

		// log chi tiet target duoc dispatch de phan tich va trace sau nay, tranh tinh trang chi biet duoc so luong target duoc dispatch ma khong biet duoc target nao va duoc dispatch nhu the nao
		uc.l.Infof(
			ctx,
			"execution.usecase.DispatchDueTargets.dispatched: source_id=%s target_id=%s scheduled_job_id=%s status=%s task_count=%d published_count=%d failed_count=%d",
			dueTarget.Source.ID,
			dueTarget.Target.ID,
			dispatchOutput.ScheduledJobID,
			dispatchOutput.Status,
			dispatchOutput.TaskCount,
			dispatchOutput.PublishedCount,
			dispatchOutput.FailedCount,
		)
	}

	return output, nil
}
