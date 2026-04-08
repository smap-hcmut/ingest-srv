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
	projectsrv "ingest-srv/pkg/microservice/project"

	"golang.org/x/sync/errgroup"
)

// Run runs the consumer server.
func (s Server) Run(ctx context.Context) error {
	dataSRepo := datasourceRepo.New(s.l, s.db)
	projectSrv := projectsrv.New(s.l, s.microservice.Project.BaseURL, s.microservice.Project.TimeoutMS, s.internalKey)
	execRepo := executionRepo.New(s.l, s.db)
	uapPublisher := uapKafka.New(s.l, s.kafka)
	uapParserRepo := uapRepo.New(s.l, s.db)
	uapParserUC := uapUC.New(s.l, uapParserRepo, s.minio, s.uapBucket, uapPublisher)
	execUC := executionUC.New(s.l, execRepo, s.minio, nil, uapParserUC, projectSrv)
	dataUC := datasourceUC.New(s.l, dataSRepo, projectSrv, execUC)

	dryrunResultRepo := dryrunRepo.New(s.l, s.db)
	dryrunUseCase := dryrunUC.New(s.l, dryrunResultRepo, dataUC, s.minio, nil)
	dryrunCompletionConsumer := dryrunConsumer.NewConsumer(s.l, s.conn, dryrunUseCase)
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
