package consumer

import (
	"context"

	executionConsumer "ingest-srv/internal/execution/delivery/rabbitmq/consumer"
	executionRepo "ingest-srv/internal/execution/repository/postgre"
	executionUC "ingest-srv/internal/execution/usecase"
	"ingest-srv/internal/uap"
	uapKafka "ingest-srv/internal/uap/delivery/kafka/producer"
	uapRepo "ingest-srv/internal/uap/repository/postgre"
	uapUC "ingest-srv/internal/uap/usecase"
)

// Run runs the consumer server.
func (s Server) Run(ctx context.Context) error {
	execRepo := executionRepo.New(s.l, s.db)
	rawBatchRepo := uapRepo.New(s.l, s.db)
	var publisher uap.Publisher
	if s.kafka != nil {
		publisher = uapKafka.New(s.l, s.kafka)
	}
	parserUC := uapUC.New(s.l, rawBatchRepo, s.minio, s.uapBucket, publisher, s.uapTopic)
	execUC := executionUC.New(s.l, execRepo, s.minio, nil, parserUC)

	return executionConsumer.NewConsumer(s.l, s.conn, execUC).Consume(ctx)
}
