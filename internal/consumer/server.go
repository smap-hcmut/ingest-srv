package consumer

import (
	"context"

	datasourceRepo "ingest-srv/internal/datasource/repository/postgre"
	datasourceUC "ingest-srv/internal/datasource/usecase"
	dryrunConsumer "ingest-srv/internal/dryrun/delivery/rabbitmq/consumer"
	dryrunRepo "ingest-srv/internal/dryrun/repository/postgre"
	dryrunUC "ingest-srv/internal/dryrun/usecase"
	executionConsumer "ingest-srv/internal/execution/delivery/rabbitmq/consumer"
	executionRepo "ingest-srv/internal/execution/repository/postgre"
	executionUC "ingest-srv/internal/execution/usecase"
	uapKafka "ingest-srv/internal/uap/delivery/kafka/producer"
	uapRepo "ingest-srv/internal/uap/repository/postgre"
	uapUC "ingest-srv/internal/uap/usecase"

	"golang.org/x/sync/errgroup"
)

// Run runs the consumer server.
func (s Server) Run(ctx context.Context) error {
	dataSRepo := datasourceRepo.New(s.l, s.db)
	dataUC := datasourceUC.New(s.l, dataSRepo)

	dryrunResultRepo := dryrunRepo.New(s.l, s.db)
	dryrunUseCase := dryrunUC.New(s.l, dryrunResultRepo, dataUC, s.minio, nil)
	dryrunCompletionConsumer := dryrunConsumer.NewConsumer(s.l, s.conn, dryrunUseCase)

	uapPublisher := uapKafka.New(s.l, s.kafka)
	uapParserRepo := uapRepo.New(s.l, s.db)
	uapParserUC := uapUC.New(s.l, uapParserRepo, s.minio, s.uapBucket, uapPublisher, s.uapTopic)

	execRepo := executionRepo.New(s.l, s.db)
	execUC := executionUC.New(s.l, execRepo, s.minio, nil, uapParserUC)
	execConsumer := executionConsumer.NewConsumer(s.l, s.conn, execUC)

	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		return execConsumer.Consume(gctx)
	})
	g.Go(func() error {
		return dryrunCompletionConsumer.Consume(gctx)
	})

	return g.Wait()
}
