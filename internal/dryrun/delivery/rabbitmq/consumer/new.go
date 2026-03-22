package consumer

import (
	"ingest-srv/internal/dryrun"

	"github.com/smap-hcmut/shared-libs/go/log"
	"github.com/smap-hcmut/shared-libs/go/rabbitmq"
)

// Consumer consumes dryrun completion messages from RabbitMQ.
type Consumer struct {
	l        log.Logger
	conn     rabbitmq.IRabbitMQ
	dryrunUC dryrun.ConsumerUseCase
}

// NewConsumer creates a new dryrun completion consumer.
func NewConsumer(l log.Logger, rabbitConn rabbitmq.IRabbitMQ, dryrunUC dryrun.ConsumerUseCase) Consumer {
	return Consumer{
		l:        l,
		conn:     rabbitConn,
		dryrunUC: dryrunUC,
	}
}
