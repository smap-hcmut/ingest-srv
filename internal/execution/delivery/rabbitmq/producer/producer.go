package producer

import (
	"context"
	"fmt"

	execution "ingest-srv/internal/execution"
	executionRabbit "ingest-srv/internal/execution/delivery/rabbitmq"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/smap-hcmut/shared-libs/go/rabbitmq"
)

func (p *implProducer) PublishDispatch(ctx context.Context, input execution.PublishDispatchInput) error {
	body, err := executionRabbit.MarshalDispatchMessage(input)
	if err != nil {
		return err
	}

	writer, err := p.getWriterByQueue(input.Queue)
	if err != nil {
		return err
	}

	return writer.Publish(ctx, rabbitmq.PublishArgs{
		Exchange:   "",
		RoutingKey: input.Queue,
		Msg: amqp.Publishing{
			ContentType:  rabbitmq.ContentTypeJSON,
			DeliveryMode: amqp.Persistent,
			Body:         body,
		},
	})
}

func (p *implProducer) getWriter(queue rabbitmq.QueueArgs) (rabbitmq.IChannel, error) {
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

	return ch, nil
}

func (p *implProducer) getWriterByQueue(queueName string) (rabbitmq.IChannel, error) {
	switch queueName {
	case executionRabbit.TikTokTasksQueueName:
		if p.tikTokTasksWriter == nil {
			return nil, fmt.Errorf("rabbitmq writer is not initialized for queue %s", queueName)
		}
		return p.tikTokTasksWriter, nil
	case executionRabbit.FacebookTasksQueueName:
		if p.facebookTasksWriter == nil {
			return nil, fmt.Errorf("rabbitmq writer is not initialized for queue %s", queueName)
		}
		return p.facebookTasksWriter, nil
	case executionRabbit.YoutubeTasksQueueName:
		if p.youtubeTasksWriter == nil {
			return nil, fmt.Errorf("rabbitmq writer is not initialized for queue %s", queueName)
		}
		return p.youtubeTasksWriter, nil
	default:
		return nil, fmt.Errorf("unsupported queue %s", queueName)
	}
}
