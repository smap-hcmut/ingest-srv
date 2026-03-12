package producer

import (
	"ingest-srv/internal/uap"

	"github.com/smap-hcmut/shared-libs/go/kafka"
	"github.com/smap-hcmut/shared-libs/go/log"
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
