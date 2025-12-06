package consumer

import (
	"context"

	dispatcherConsumer "smap-collector/internal/dispatcher/delivery/rabbitmq/consumer"
	dispatcherProducer "smap-collector/internal/dispatcher/delivery/rabbitmq/producer"
	dispatcherUsecase "smap-collector/internal/dispatcher/usecase"
	resultsConsumer "smap-collector/internal/results/delivery/rabbitmq/consumer"
	resultsUsecase "smap-collector/internal/results/usecase"
	stateRedis "smap-collector/internal/state/repository/redis"
	stateUsecase "smap-collector/internal/state/usecase"
	webhookUsecase "smap-collector/internal/webhook/usecase"
	"smap-collector/pkg/project"
)

func (srv *Server) Run(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}

	srv.l.Info(ctx, "consumer starting")

	// 1. Init Producers
	prod := dispatcherProducer.New(srv.l, srv.conn)
	if err := prod.Run(); err != nil {
		return err
	}

	// 2. Init Microservice
	projectClient := project.NewClient(srv.cfg.ProjectConfig, srv.l)

	// 3. Init UseCases
	stateRepo := stateRedis.NewRedisRepository(srv.l, srv.cfg.RedisClient)
	stateUC := stateUsecase.NewUseCase(srv.l, stateRepo, srv.cfg.StateOptions)
	webhookUC := webhookUsecase.NewUseCase(srv.l, projectClient)
	dispatcherUC := dispatcherUsecase.NewUseCaseWithDeps(srv.l, prod, srv.cfg.DispatcherOptions, stateUC, webhookUC)
	resultsUC := resultsUsecase.NewUseCase(srv.l, projectClient, stateUC, webhookUC)
	dispatchC := dispatcherConsumer.NewConsumer(srv.l, srv.conn, dispatcherUC)
	resultsC := resultsConsumer.NewConsumer(srv.l, srv.conn, resultsUC)

	dispatchC.Consume()
	srv.l.Info(ctx, "Dispatcher consumer started (collector.inbound.tasks)")

	dispatchC.ConsumeProjectEvents()
	srv.l.Info(ctx, "Dispatcher consumer started (smap.events.project.created)")

	resultsC.Consume()
	srv.l.Info(ctx, "Dispatcher consumer started (results.inbound.data)")
	srv.l.Info(ctx, "All consumers started")
	<-ctx.Done()

	return nil
}

// Close releases MQ resources.
func (srv *Server) Close() {
	if srv.conn != nil {
		srv.conn.Close()
	}
}
