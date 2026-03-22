package producer

import (
	"context"

	"ingest-srv/internal/dryrun"

	"github.com/smap-hcmut/shared-libs/go/log"
	"github.com/smap-hcmut/shared-libs/go/rabbitmq"
)

type Producer interface {
	PublishDispatch(ctx context.Context, input dryrun.PublishDispatchInput) error
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

func New(l log.Logger, rabbitConn rabbitmq.IRabbitMQ) Producer {
	return &implProducer{
		l:    l,
		conn: rabbitConn,
	}
}
