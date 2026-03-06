package main

import (
	"context"
	"fmt"

	"ingest-srv/config"
	configKafka "ingest-srv/config/kafka"
	configMinio "ingest-srv/config/minio"
	configPostgre "ingest-srv/config/postgre"
	configRabbit "ingest-srv/config/rabbitmq"
	configRedis "ingest-srv/config/redis"
	"ingest-srv/internal/scheduler"
	"ingest-srv/pkg/discord"
	"ingest-srv/pkg/log"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Println("Failed to load config:", err)
		return
	}

	logger := log.Init(log.ZapConfig{
		Level:        cfg.Logger.Level,
		Mode:         cfg.Logger.Mode,
		Encoding:     cfg.Logger.Encoding,
		ColorEnabled: cfg.Logger.ColorEnabled,
	})

	ctx := context.Background()

	var discordClient discord.IDiscord
	if cfg.Discord.WebhookURL != "" {
		discordClient, err = discord.New(logger, cfg.Discord.WebhookURL)
		if err != nil {
			logger.Warnf(ctx, "Discord webhook is invalid, running without discord: %v", err)
			discordClient = nil
		} else {
			defer discordClient.Close()
		}
	}

	// Initialize infrastructure dependencies in best-effort mode.
	if db, e := configPostgre.Connect(ctx, cfg.Postgres); e != nil {
		logger.Warnf(ctx, "Scheduler PostgreSQL init failed: %v", e)
	} else {
		defer configPostgre.Disconnect(ctx, db)
	}

	if redisClient, e := configRedis.Connect(ctx, cfg.Redis); e != nil {
		logger.Warnf(ctx, "Scheduler Redis init failed: %v", e)
	} else {
		_ = redisClient
		defer configRedis.Disconnect()
	}

	if minioClient, e := configMinio.Connect(ctx, &cfg.MinIO); e != nil {
		logger.Warnf(ctx, "Scheduler MinIO init failed: %v", e)
	} else {
		_ = minioClient
		defer configMinio.Disconnect()
	}

	if producer, e := configKafka.ConnectProducer(cfg.Kafka); e != nil {
		logger.Warnf(ctx, "Scheduler Kafka init failed: %v", e)
	} else {
		_ = producer
		defer configKafka.DisconnectProducer()
	}

	if conn, e := configRabbit.Connect(cfg.RabbitMQ); e != nil {
		logger.Warnf(ctx, "Scheduler RabbitMQ init failed: %v", e)
	} else {
		_ = conn
		defer configRabbit.Disconnect()
	}

	s, err := scheduler.New(scheduler.Config{
		Logger:  logger,
		Config:  cfg.Scheduler,
		Discord: discordClient,
	})
	if err != nil {
		logger.Error(ctx, "Failed to initialize scheduler: ", err)
		return
	}

	if err := s.Start(); err != nil {
		logger.Error(ctx, "Scheduler stopped with error: ", err)
		return
	}
}
