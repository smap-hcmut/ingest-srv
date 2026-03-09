package producer

import (
	"ingest-srv/internal/uap"
	"ingest-srv/pkg/kafka"
	"ingest-srv/pkg/log"
)

type publisher struct {
	logger   log.Logger
	producer kafka.IProducer
}

var _ uap.Publisher = (*publisher)(nil)

// New creates a new Kafka UAP publisher.
func New(logger log.Logger, producer kafka.IProducer) uap.Publisher {
	return &publisher{
		logger:   logger,
		producer: producer,
	}
}
