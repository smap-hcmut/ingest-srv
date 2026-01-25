package producer

import (
	"context"
	"encoding/json"
	"errors"

	rabb "smap-collector/internal/dispatcher/delivery/rabbitmq"
	"smap-collector/internal/models"
	pkgRabbit "smap-collector/pkg/rabbitmq"
)

func (p implProducer) PublishTikTokTask(ctx context.Context, task models.TikTokCollectorTask) error {
	if p.writer == nil {
		return errors.New("producer not started")
	}

	body, err := json.Marshal(task)
	if err != nil {
		return err
	}

	return p.writer.Publish(ctx, pkgRabbit.PublishArgs{
		Exchange:   rabb.ExchangeTikTok,
		RoutingKey: rabb.RoutingKeyTikTok,
		Msg: pkgRabbit.Publishing{
			Body:        body,
			ContentType: pkgRabbit.ContentTypeJSON,
		},
	})
}

func (p implProducer) PublishYouTubeTask(ctx context.Context, task models.YouTubeCollectorTask) error {
	if p.writer == nil {
		return errors.New("producer not started")
	}

	body, err := json.Marshal(task)
	if err != nil {
		return err
	}

	return p.writer.Publish(ctx, pkgRabbit.PublishArgs{
		Exchange:   rabb.ExchangeYouTube,
		RoutingKey: rabb.RoutingKeyYouTube,
		Msg: pkgRabbit.Publishing{
			Body:        body,
			ContentType: pkgRabbit.ContentTypeJSON,
		},
	})
}
