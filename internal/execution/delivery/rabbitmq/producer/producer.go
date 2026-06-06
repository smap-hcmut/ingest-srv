package producer

import (
	"context"
	"fmt"

	execution "ingest-srv/internal/execution"
	executionRabbit "ingest-srv/internal/execution/delivery/rabbitmq"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/smap-hcmut/shared-libs/go/constants"
	"github.com/smap-hcmut/shared-libs/go/rabbitmq"
)

func (p *implProducer) PublishDispatch(ctx context.Context, input execution.PublishDispatchInput) error {
	body, err := executionRabbit.MarshalDispatchMessage(input)
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
		p.l.Warnf(ctx, "execution.rabbitmq.PublishDispatch.publish_failed_rebuild: queue=%s err=%v", input.Queue, err)
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

func (p *implProducer) getWriterWithQueue(exchange rabbitmq.ExchangeArgs, queue rabbitmq.QueueArgs, routingKey string) (rabbitmq.IChannel, error) {
	if p.conn == nil {
		return nil, fmt.Errorf("rabbitmq connection is not initialized")
	}

	ch, err := p.conn.Channel()
	if err != nil {
		return nil, err
	}

	if err := p.declarePublishTopology(ch, exchange, queue, routingKey); err != nil {
		_ = ch.Close()
		return nil, err
	}

	return ch, nil
}

func (p *implProducer) declarePublishTopology(ch rabbitmq.IChannel, exchange rabbitmq.ExchangeArgs, queue rabbitmq.QueueArgs, routingKey string) error {
	if ch == nil {
		return fmt.Errorf("rabbitmq channel is not initialized")
	}
	// Declare DLX topology first so the main queue's x-dead-letter-exchange
	// reference resolves at queue-declare time. RabbitMQ requires the target
	// exchange to exist; otherwise nack(requeue=false) drops silently.
	if err := p.declareDLXTopology(ch, queue.Name, routingKey); err != nil {
		return err
	}
	if err := ch.ExchangeDeclare(exchange); err != nil {
		return err
	}
	if _, err := ch.QueueDeclare(queue); err != nil {
		return err
	}
	if err := ch.QueueBind(rabbitmq.QueueBindArgs{
		Queue:      queue.Name,
		Exchange:   exchange.Name,
		RoutingKey: routingKey,
	}); err != nil {
		return err
	}

	return nil
}

// declareDLXTopology declares the shared dead-letter exchange and the matching
// DLQ for the given task queue. Idempotent; safe to invoke on every publish
// session bootstrap. Returns an error if the platform's DLQ is unknown.
func (p *implProducer) declareDLXTopology(ch rabbitmq.IChannel, taskQueueName, routingKey string) error {
	if err := ch.ExchangeDeclare(executionRabbit.DLXExchange); err != nil {
		return err
	}

	var dlq rabbitmq.QueueArgs
	switch taskQueueName {
	case executionRabbit.TikTokTasksQueue.Name:
		dlq = executionRabbit.TikTokTasksDLQQueue
	case executionRabbit.FacebookTasksQueue.Name:
		dlq = executionRabbit.FacebookTasksDLQQueue
	case executionRabbit.YoutubeTasksQueue.Name:
		dlq = executionRabbit.YouTubeTasksDLQQueue
	default:
		return nil
	}

	if _, err := ch.QueueDeclare(dlq); err != nil {
		return err
	}
	return ch.QueueBind(rabbitmq.QueueBindArgs{
		Queue:      dlq.Name,
		Exchange:   executionRabbit.DLXExchange.Name,
		RoutingKey: routingKey,
	})
}

func (p *implProducer) ensurePublishRouteLocked(ctx context.Context, queueName execution.QueueName) (rabbitmq.IChannel, rabbitmq.ExchangeArgs, string, error) {
	writer, exchange, queue, routingKey, err := p.getPublishRouteConfigByQueue(queueName)
	if err != nil {
		return nil, rabbitmq.ExchangeArgs{}, "", err
	}
	if writer != nil {
		if err := p.declarePublishTopology(writer, exchange, queue, routingKey); err == nil {
			return writer, exchange, routingKey, nil
		} else {
			p.l.Warnf(ctx, "execution.rabbitmq.ensurePublishRoute.rebuild: queue=%s err=%v", queueName, err)
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

func (p *implProducer) getPublishRouteConfigByQueue(queueName execution.QueueName) (rabbitmq.IChannel, rabbitmq.ExchangeArgs, rabbitmq.QueueArgs, string, error) {
	switch queueName {
	case execution.QueueName(constants.QueueTikTokTasks):
		return p.tikTokTasksWriter, executionRabbit.TikTokTasksExchange, executionRabbit.TikTokTasksQueue, executionRabbit.TikTokTasksRoutingKey, nil
	case execution.QueueName(constants.QueueFacebookTasks):
		return p.facebookTasksWriter, executionRabbit.FacebookTasksExchange, executionRabbit.FacebookTasksQueue, executionRabbit.FacebookTasksRoutingKey, nil
	case execution.QueueName(constants.QueueYouTubeTasks):
		return p.youtubeTasksWriter, executionRabbit.YoutubeTasksExchange, executionRabbit.YoutubeTasksQueue, executionRabbit.YoutubeTasksRoutingKey, nil
	default:
		return nil, rabbitmq.ExchangeArgs{}, rabbitmq.QueueArgs{}, "", fmt.Errorf("unsupported queue %s", queueName)
	}
}

func (p *implProducer) setWriterLocked(queueName execution.QueueName, writer rabbitmq.IChannel) {
	switch queueName {
	case execution.QueueName(constants.QueueTikTokTasks):
		p.tikTokTasksWriter = writer
	case execution.QueueName(constants.QueueFacebookTasks):
		p.facebookTasksWriter = writer
	case execution.QueueName(constants.QueueYouTubeTasks):
		p.youtubeTasksWriter = writer
	}
}

func (p *implProducer) clearWriterLocked(queueName execution.QueueName) {
	switch queueName {
	case execution.QueueName(constants.QueueTikTokTasks):
		p.tikTokTasksWriter = nil
	case execution.QueueName(constants.QueueFacebookTasks):
		p.facebookTasksWriter = nil
	case execution.QueueName(constants.QueueYouTubeTasks):
		p.youtubeTasksWriter = nil
	}
}
