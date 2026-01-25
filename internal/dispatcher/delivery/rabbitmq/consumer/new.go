package consumer

import (
	"smap-collector/internal/dispatcher"
	pkgLog "smap-collector/pkg/log"
	pkgRabbit "smap-collector/pkg/rabbitmq"
)

type Consumer struct {
	l    pkgLog.Logger
	conn *pkgRabbit.Connection
	uc   dispatcher.UseCase
}

// NewConsumer creates a new consumer.
func NewConsumer(l pkgLog.Logger, conn *pkgRabbit.Connection, uc dispatcher.UseCase) Consumer {
	return Consumer{
		l:    l,
		conn: conn,
		uc:   uc,
	}
}
