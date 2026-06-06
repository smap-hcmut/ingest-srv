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

	p.mu.Lock()
	defer p.mu.Unlock()

	writer, exchange, routingKey, err := p.ensurePublishRouteLocked(ctx, input.Queue)
	if err != nil {
		return err
	}

	publishArgs := rabbitmq.PublishArgs{
		Exchange:   exchange.Name,
		RoutingKey: routingKey,
		Msg: amqp.Publishing{
			ContentType:  rabbitmq.ContentTypeJSON,
			DeliveryMode: amqp.Persistent,
			Body:         body,
		},
	}

	if err := writer.Publish(ctx, publishArgs); err != nil {
		p.l.Warnf(ctx, "dryrun.rabbitmq.PublishDispatch.publish_failed_rebuild: queue=%s err=%v", input.Queue, err)
		_ = writer.Close()
		p.clearWriterLocked(input.Queue)

		writer, exchange, routingKey, rebuildErr := p.ensurePublishRouteLocked(ctx, input.Queue)
		if rebuildErr != nil {
			return fmt.Errorf("publish dispatch failed: %w; rebuild route: %v", err, rebuildErr)
		}
		publishArgs.Exchange = exchange.Name
		publishArgs.RoutingKey = routingKey
		return writer.Publish(ctx, publishArgs)
	}

	return nil
}

func (p *implProducer) ensurePublishRouteLocked(ctx context.Context, queueName dryrun.QueueName) (rabbitmq.IChannel, rabbitmq.ExchangeArgs, string, error) {
	writer, exchange, queue, routingKey, err := p.getPublishRouteConfigByQueue(queueName)
	if err != nil {
		return nil, rabbitmq.ExchangeArgs{}, "", err
	}
	if writer != nil {
		if err := p.declarePublishTopology(writer, exchange, queue, routingKey); err == nil {
			return writer, exchange, routingKey, nil
		} else {
			p.l.Warnf(ctx, "dryrun.rabbitmq.ensurePublishRoute.rebuild: queue=%s err=%v", queueName, err)
			_ = writer.Close()
			p.clearWriterLocked(queueName)
		}
	}

	writer, err = p.getWriterWithQueue(exchange, queue, routingKey)
	if err != nil {
		return nil, rabbitmq.ExchangeArgs{}, "", err
	}
	p.setWriterLocked(queueName, writer)
	return writer, exchange, routingKey, nil
}

func (p *implProducer) getPublishRouteConfigByQueue(queueName dryrun.QueueName) (rabbitmq.IChannel, rabbitmq.ExchangeArgs, rabbitmq.QueueArgs, string, error) {
	switch queueName {
	case dryrun.QueueName(constants.QueueTikTokTasks):
		return p.tikTokTasksWriter, dryrunRabbit.TikTokTasksExchange, dryrunRabbit.TikTokTasksQueue, dryrunRabbit.TikTokTasksRoutingKey, nil
	case dryrun.QueueName(constants.QueueFacebookTasks):
		return p.facebookTasksWriter, dryrunRabbit.FacebookTasksExchange, dryrunRabbit.FacebookTasksQueue, dryrunRabbit.FacebookTasksRoutingKey, nil
	case dryrun.QueueName(constants.QueueYouTubeTasks):
		return p.youtubeTasksWriter, dryrunRabbit.YoutubeTasksExchange, dryrunRabbit.YoutubeTasksQueue, dryrunRabbit.YoutubeTasksRoutingKey, nil
	default:
		return nil, rabbitmq.ExchangeArgs{}, rabbitmq.QueueArgs{}, "", fmt.Errorf("unsupported queue %s", queueName)
	}
}

func (p *implProducer) setWriterLocked(queueName dryrun.QueueName, writer rabbitmq.IChannel) {
	switch queueName {
	case dryrun.QueueName(constants.QueueTikTokTasks):
		p.tikTokTasksWriter = writer
	case dryrun.QueueName(constants.QueueFacebookTasks):
		p.facebookTasksWriter = writer
	case dryrun.QueueName(constants.QueueYouTubeTasks):
		p.youtubeTasksWriter = writer
	}
}

func (p *implProducer) clearWriterLocked(queueName dryrun.QueueName) {
	switch queueName {
	case dryrun.QueueName(constants.QueueTikTokTasks):
		p.tikTokTasksWriter = nil
	case dryrun.QueueName(constants.QueueFacebookTasks):
		p.facebookTasksWriter = nil
	case dryrun.QueueName(constants.QueueYouTubeTasks):
		p.youtubeTasksWriter = nil
	}
}
