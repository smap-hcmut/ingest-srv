package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"ingest-srv/config"
	configKafka "ingest-srv/config/kafka"
	configMinio "ingest-srv/config/minio"
	configPostgre "ingest-srv/config/postgre"
	configRabbit "ingest-srv/config/rabbitmq"
	"ingest-srv/internal/consumer"

	"github.com/smap-hcmut/shared-libs/go/log"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}

	logger := log.NewZapLogger(log.ZapConfig{
		Level:        cfg.Logger.Level,
		Mode:         cfg.Logger.Mode,
		Encoding:     cfg.Logger.Encoding,
		ColorEnabled: cfg.Logger.ColorEnabled,
	})

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	postgresDB, err := configPostgre.Connect(ctx, cfg.Postgres)
	if err != nil {
		logger.Errorf(ctx, "PostgreSQL connect failed: %v", err)
		postgresDB = nil
	} else {
		defer configPostgre.Disconnect(ctx, postgresDB)
	}

	minioClient, err := configMinio.Connect(ctx, &cfg.MinIO)
	if err != nil {
		logger.Errorf(ctx, "MinIO connect failed: %v", err)
		minioClient = nil
	} else {
		defer configMinio.Disconnect()
	}

	rabbitConn, err := configRabbit.Connect(cfg.RabbitMQ)
	if err != nil {
		logger.Errorf(ctx, "RabbitMQ connect failed: %v", err)
		rabbitConn = nil
	} else {
		defer configRabbit.Disconnect()
	}

	kafkaCfg := cfg.Kafka
	kafkaCfg.Topic = cfg.Kafka.UAPTopic
	kafkaProducer, err := configKafka.ConnectProducer(kafkaCfg)
	if err != nil {
		logger.Warnf(ctx, "Kafka producer connect failed, UAP publish will be disabled: %v", err)
		kafkaProducer = nil
	} else {
		defer configKafka.DisconnectProducer()
	}

	if postgresDB == nil || minioClient == nil || rabbitConn == nil {
		logger.Errorf(ctx, "Execution completion consumer requires postgres, minio, and rabbitmq")
		os.Exit(1)
	}

	consumerSrv := consumer.NewServer(logger, consumer.ServerConfig{
		Conn:      rabbitConn,
		DB:        postgresDB,
		MinIO:     minioClient,
		UAPBucket: cfg.MinIO.Bucket,
		Kafka:     kafkaProducer,
		UAPTopic:  cfg.Kafka.UAPTopic,
	})

	if err := consumerSrv.Run(ctx); err != nil {
		logger.Errorf(ctx, "Execution completion consumer error: %v", err)
		os.Exit(1)
	}
}
