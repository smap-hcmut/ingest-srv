package consumer

import (
	"ingest-srv/internal/execution"
	"ingest-srv/pkg/log"
	"ingest-srv/pkg/rabbitmq"
)

// Consumer consumes execution completion messages from RabbitMQ.
type Consumer struct {
	l    log.Logger
	conn rabbitmq.IRabbitMQ
	uc   execution.UseCase
}

// NewConsumer creates a new execution completion consumer.
func NewConsumer(l log.Logger, rabbitConn rabbitmq.IRabbitMQ, uc execution.UseCase) Consumer {
	return Consumer{
		l:    l,
		conn: rabbitConn,
		uc:   uc,
	}
}
