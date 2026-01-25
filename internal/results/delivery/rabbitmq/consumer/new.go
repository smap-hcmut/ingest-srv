package consumer

import (
	"smap-collector/internal/results"
	pkgLog "smap-collector/pkg/log"
	pkgRabbit "smap-collector/pkg/rabbitmq"
)

type Consumer struct {
	l    pkgLog.Logger
	conn *pkgRabbit.Connection
	uc   results.UseCase
}

// NewConsumer creates a new results consumer.
func NewConsumer(l pkgLog.Logger, conn *pkgRabbit.Connection, uc results.UseCase) Consumer {
	return Consumer{
		l:    l,
		conn: conn,
		uc:   uc,
	}
}
