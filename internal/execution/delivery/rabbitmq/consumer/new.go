package consumer

import (
	"ingest-srv/internal/execution"

	"github.com/smap-hcmut/shared-libs/go/log"
	"github.com/smap-hcmut/shared-libs/go/rabbitmq"
)

// Consumer consumes execution completion messages from RabbitMQ.
type Consumer struct {
	l      log.Logger
	conn   rabbitmq.IRabbitMQ
	execUC execution.ConsumerUseCase
}

// NewConsumer creates a new execution completion consumer.
func NewConsumer(l log.Logger, rabbitConn rabbitmq.IRabbitMQ, execUC execution.ConsumerUseCase) Consumer {
	return Consumer{
		l:      l,
		conn:   rabbitConn,
		execUC: execUC,
	}
}
