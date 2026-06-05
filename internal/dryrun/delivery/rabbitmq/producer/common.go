package producer

import (
	"fmt"

	dryrunRabbit "ingest-srv/internal/dryrun/delivery/rabbitmq"

	"github.com/smap-hcmut/shared-libs/go/rabbitmq"
)

func (p *implProducer) Run() error {
	p.mu.Lock()
	defer p.mu.Unlock()

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
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.tikTokTasksWriter != nil {
		_ = p.tikTokTasksWriter.Close()
		p.tikTokTasksWriter = nil
	}
	if p.facebookTasksWriter != nil {
		_ = p.facebookTasksWriter.Close()
		p.facebookTasksWriter = nil
	}
	if p.youtubeTasksWriter != nil {
		_ = p.youtubeTasksWriter.Close()
		p.youtubeTasksWriter = nil
	}
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
