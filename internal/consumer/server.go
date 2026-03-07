package consumer

import (
	"context"

	executionConsumer "ingest-srv/internal/execution/delivery/rabbitmq/consumer"
	executionRepo "ingest-srv/internal/execution/repository/postgre"
	executionUC "ingest-srv/internal/execution/usecase"
)

// Run runs the consumer server.
func (s Server) Run(ctx context.Context) error {
	execRepo := executionRepo.New(s.l, s.db)
	execUC := executionUC.New(s.l, execRepo, s.minio, nil)

	return executionConsumer.NewConsumer(s.l, s.conn, execUC).Consume(ctx)
}
