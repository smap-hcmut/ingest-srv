package producer

import (
	dryrunRabbit "ingest-srv/internal/dryrun/delivery/rabbitmq"

	"github.com/smap-hcmut/shared-libs/go/rabbitmq"
)

func (p *implProducer) Run() error {
	var err error

	p.tikTokTasksWriter, err = p.getWriterWithQueue(
		dryrunRabbit.TikTokTasksExchange,
		dryrunRabbit.TikTokTasksQueue,
		dryrunRabbit.TikTokTasksRoutingKey,
	)
	if err != nil {
		return err
	}

	p.facebookTasksWriter, err = p.getWriterWithQueue(
		dryrunRabbit.FacebookTasksExchange,
		dryrunRabbit.FacebookTasksQueue,
		dryrunRabbit.FacebookTasksRoutingKey,
	)
	if err != nil {
		p.Close()
		return err
	}

	p.youtubeTasksWriter, err = p.getWriterWithQueue(
		dryrunRabbit.YoutubeTasksExchange,
		dryrunRabbit.YoutubeTasksQueue,
		dryrunRabbit.YoutubeTasksRoutingKey,
	)
	if err != nil {
		p.Close()
		return err
	}

	return nil
}

func (p *implProducer) Close() {
	if p.tikTokTasksWriter != nil {
		_ = p.tikTokTasksWriter.Close()
	}
	if p.facebookTasksWriter != nil {
		_ = p.facebookTasksWriter.Close()
	}
	if p.youtubeTasksWriter != nil {
		_ = p.youtubeTasksWriter.Close()
	}
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
