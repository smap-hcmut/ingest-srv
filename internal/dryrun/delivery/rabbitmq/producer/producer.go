package producer

import (
	"context"
	"fmt"

	"ingest-srv/internal/dryrun"
	dryrunRabbit "ingest-srv/internal/dryrun/delivery/rabbitmq"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/smap-hcmut/shared-libs/go/constants"
	"github.com/smap-hcmut/shared-libs/go/rabbitmq"
)

func (p *implProducer) PublishDispatch(ctx context.Context, input dryrun.PublishDispatchInput) error {
	body, err := dryrunRabbit.MarshalDispatchMessage(input)
	if err != nil {
		return err
	}

	writer, exchange, routingKey, err := p.getPublishRouteByQueue(input.Queue)
	if err != nil {
		return err
	}

	return writer.Publish(ctx, rabbitmq.PublishArgs{
		Exchange:   exchange,
		RoutingKey: routingKey,
		Msg: amqp.Publishing{
			ContentType:  rabbitmq.ContentTypeJSON,
			DeliveryMode: amqp.Persistent,
			Body:         body,
		},
	})
}

func (p *implProducer) getPublishRouteByQueue(queueName dryrun.QueueName) (rabbitmq.IChannel, string, string, error) {
	switch queueName {
	case dryrun.QueueName(constants.QueueTikTokTasks):
		if p.tikTokTasksWriter == nil {
			return nil, "", "", fmt.Errorf("rabbitmq writer is not initialized for queue %s", queueName)
		}
		return p.tikTokTasksWriter, constants.ExchangeTikTokTasks, dryrunRabbit.TikTokTasksRoutingKey, nil
	case dryrun.QueueName(constants.QueueFacebookTasks):
		if p.facebookTasksWriter == nil {
			return nil, "", "", fmt.Errorf("rabbitmq writer is not initialized for queue %s", queueName)
		}
		return p.facebookTasksWriter, constants.ExchangeFacebookTasks, dryrunRabbit.FacebookTasksRoutingKey, nil
	case dryrun.QueueName(constants.QueueYouTubeTasks):
		if p.youtubeTasksWriter == nil {
			return nil, "", "", fmt.Errorf("rabbitmq writer is not initialized for queue %s", queueName)
		}
		return p.youtubeTasksWriter, constants.ExchangeYouTubeTasks, dryrunRabbit.YoutubeTasksRoutingKey, nil
	default:
		return nil, "", "", fmt.Errorf("unsupported queue %s", queueName)
	}
}
