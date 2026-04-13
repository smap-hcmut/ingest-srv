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

func (p *implProducer) getWriterWithQueue(exchange rabbitmq.ExchangeArgs, queue rabbitmq.QueueArgs, routingKey string) (rabbitmq.IChannel, error) {
	if p.conn == nil {
		return nil, nil
	}

	ch, err := p.conn.Channel()
	if err != nil {
		return nil, err
	}

	if _, err := ch.QueueDeclare(queue); err != nil {
		_ = ch.Close()
		return nil, err
	}

	if err := ch.ExchangeDeclare(exchange); err != nil {
		_ = ch.Close()
		return nil, err
	}

	if err := ch.QueueBind(rabbitmq.QueueBindArgs{
		Queue:      queue.Name,
		Exchange:   exchange.Name,
		RoutingKey: routingKey,
	}); err != nil {
		_ = ch.Close()
		return nil, err
	}

	return ch, nil
}

func (p *implProducer) getPublishRouteByQueue(queueName execution.QueueName) (rabbitmq.IChannel, string, string, error) {
	switch queueName {
	case execution.QueueName(constants.QueueTikTokTasks):
		if p.tikTokTasksWriter == nil {
			return nil, "", "", fmt.Errorf("rabbitmq writer is not initialized for queue %s", queueName)
		}
		return p.tikTokTasksWriter, constants.ExchangeTikTokTasks, executionRabbit.TikTokTasksRoutingKey, nil
	case execution.QueueName(constants.QueueFacebookTasks):
		if p.facebookTasksWriter == nil {
			return nil, "", "", fmt.Errorf("rabbitmq writer is not initialized for queue %s", queueName)
		}
		return p.facebookTasksWriter, constants.ExchangeFacebookTasks, executionRabbit.FacebookTasksRoutingKey, nil
	case execution.QueueName(constants.QueueYouTubeTasks):
		if p.youtubeTasksWriter == nil {
			return nil, "", "", fmt.Errorf("rabbitmq writer is not initialized for queue %s", queueName)
		}
		return p.youtubeTasksWriter, constants.ExchangeYouTubeTasks, executionRabbit.YoutubeTasksRoutingKey, nil
	default:
		return nil, "", "", fmt.Errorf("unsupported queue %s", queueName)
	}
}
