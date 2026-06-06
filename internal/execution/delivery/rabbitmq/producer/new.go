package producer

import (
	"context"
	"sync"

	execution "ingest-srv/internal/execution"
	executionRabbit "ingest-srv/internal/execution/delivery/rabbitmq"

	"github.com/smap-hcmut/shared-libs/go/log"
	"github.com/smap-hcmut/shared-libs/go/rabbitmq"
)

// Producer defines the RabbitMQ publisher for execution dispatches.
type Producer interface {
	PublishDispatch(ctx context.Context, input execution.PublishDispatchInput) error
	Run() error
	Close()
}

type implProducer struct {
	l                   log.Logger
	conn                rabbitmq.IRabbitMQ
	mu                  sync.Mutex
	tikTokTasksWriter   rabbitmq.IChannel
	facebookTasksWriter rabbitmq.IChannel
	youtubeTasksWriter  rabbitmq.IChannel
}

// New creates a new execution dispatch producer.
func New(l log.Logger, rabbitConn rabbitmq.IRabbitMQ) Producer {
	return &implProducer{
		l:    l,
		conn: rabbitConn,
	}
}

func (p *implProducer) Run() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	var err error

	p.tikTokTasksWriter, err = p.getWriterWithQueue(
		executionRabbit.TikTokTasksExchange,
		executionRabbit.TikTokTasksQueue,
		executionRabbit.TikTokTasksRoutingKey,
	)
	if err != nil {
		return err
	}

	p.facebookTasksWriter, err = p.getWriterWithQueue(
		executionRabbit.FacebookTasksExchange,
		executionRabbit.FacebookTasksQueue,
		executionRabbit.FacebookTasksRoutingKey,
	)
	if err != nil {
		p.Close()
		return err
	}

	p.youtubeTasksWriter, err = p.getWriterWithQueue(
		executionRabbit.YoutubeTasksExchange,
		executionRabbit.YoutubeTasksQueue,
		executionRabbit.YoutubeTasksRoutingKey,
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
