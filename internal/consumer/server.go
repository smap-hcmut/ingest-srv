package consumer

import (
	"context"
	"time"
)

// Run starts consumer lifecycle and blocks until context is canceled.
func (srv *ConsumerServer) Run(ctx context.Context) error {
	srv.l.Info(ctx, "Starting Ingest Consumer Service...")

	if srv.kafkaConsumer == nil {
		srv.l.Warn(ctx, "Kafka consumer not initialized, consumer is running in idle mode")
		<-ctx.Done()
		srv.l.Info(ctx, "Consumer service stopped")
		return nil
	}

	handler := NewNoopHandler(srv.l)
	topics := []string{srv.kafkaConfig.Topic}

	go func() {
		for {
			err := srv.kafkaConsumer.ConsumeWithContext(ctx, topics, handler)
			if err != nil && ctx.Err() == nil {
				srv.l.Errorf(ctx, "Kafka consume error: %v", err)
				time.Sleep(2 * time.Second)
				continue
			}
			if ctx.Err() != nil {
				return
			}
		}
	}()

	<-ctx.Done()
	srv.l.Info(context.Background(), "Shutdown signal received, closing consumer...")

	if err := srv.kafkaConsumer.Close(); err != nil {
		srv.l.Errorf(context.Background(), "Failed to close kafka consumer: %v", err)
	}

	srv.l.Info(context.Background(), "Consumer service stopped gracefully")
	return nil
}
