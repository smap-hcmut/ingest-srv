package usecase

import (
	"context"

	"smap-collector/internal/dispatcher"
	rabb "smap-collector/internal/dispatcher/delivery/rabbitmq"
	"smap-collector/internal/models"
)

func (uc implUseCase) Dispatch(ctx context.Context, req models.CrawlRequest) ([]models.CollectorTask, error) {
	// Validate request upfront
	if err := uc.validateRequest(&req, uc.defaultOptions); err != nil {
		return nil, err
	}

	targetPlatforms := uc.selectPlatforms()
	if len(targetPlatforms) == 0 {
		return nil, dispatcher.ErrUnknownRoute
	}

	tasks := make([]models.CollectorTask, 0, len(targetPlatforms))
	for _, platform := range targetPlatforms {
		baseTask := models.BaseCollectorTask{
			JobID:         req.JobID,
			Platform:      platform,
			TaskType:      req.TaskType,
			TimeRange:     req.TimeRange,
			Attempt:       req.Attempt,
			MaxAttempts:   req.MaxAttempts,
			SchemaVersion: uc.defaultOptions.SchemaVersion,
			EmittedAt:     req.EmittedAt,
			Headers: map[string]any{
				"x-schema-version": uc.defaultOptions.SchemaVersion,
			},
		}

		payload, err := uc.mapPayload(platform, req.TaskType, req.Payload)
		if err != nil {
			return nil, err
		}

		var errPublish error
		var task models.CollectorTask

		switch platform {
		case models.PlatformTikTok:
			tkTask := models.TikTokCollectorTask{
				BaseCollectorTask: baseTask,
				Payload:           payload,
			}
			tkTask.RoutingKey = rabb.RoutingKeyTikTok
			task = tkTask
			errPublish = uc.PublishTikTokTask(ctx, tkTask)
		case models.PlatformYouTube:
			ytTask := models.YouTubeCollectorTask{
				BaseCollectorTask: baseTask,
				Payload:           payload,
			}
			ytTask.RoutingKey = rabb.RoutingKeyYouTube
			task = ytTask
			errPublish = uc.PublishYouTubeTask(ctx, ytTask)
		default:
			// Skip unknown platforms or handle error
			continue
		}

		if errPublish != nil {
			uc.l.Errorf(ctx, "dispatcher.usecase.dispatch: publish failed for platform=%s, job_id=%s: %v", platform, req.JobID, errPublish)
			return nil, ErrPublishFailed
		}

		tasks = append(tasks, task)
	}

	return tasks, nil
}
