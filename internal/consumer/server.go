package consumer

import (
	"context"

	executionConsumer "ingest-srv/internal/execution/delivery/rabbitmq/consumer"
	executionRepo "ingest-srv/internal/execution/repository/postgre"
	executionUC "ingest-srv/internal/execution/usecase"
	uapRepo "ingest-srv/internal/uap/repository/postgre"
	uapUC "ingest-srv/internal/uap/usecase"
)

// Run runs the consumer server.
func (s Server) Run(ctx context.Context) error {
	execRepo := executionRepo.New(s.l, s.db)
	rawBatchRepo := uapRepo.New(s.l, s.db)
	parserUC := uapUC.New(s.l, rawBatchRepo, s.minio, s.uapBucket)
	execUC := executionUC.New(s.l, execRepo, s.minio, nil, parserUC)

	return executionConsumer.NewConsumer(s.l, s.conn, execUC).Consume(ctx)
}
