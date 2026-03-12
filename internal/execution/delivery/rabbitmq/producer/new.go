package producer

import (
	"context"

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
	var err error

	p.tikTokTasksWriter, err = p.getWriter(executionRabbit.TikTokTasksQueue)
	if err != nil {
		return err
	}

	p.facebookTasksWriter, err = p.getWriter(executionRabbit.FacebookTasksQueue)
	if err != nil {
		p.Close()
		return err
	}

	p.youtubeTasksWriter, err = p.getWriter(executionRabbit.YoutubeTasksQueue)
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
