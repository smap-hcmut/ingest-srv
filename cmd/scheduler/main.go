package main

import (
	"context"
	"fmt"
	"os"

	"ingest-srv/config"
	configPostgre "ingest-srv/config/postgre"
	configRabbit "ingest-srv/config/rabbitmq"
	"ingest-srv/internal/scheduler"

	"github.com/smap-hcmut/shared-libs/go/discord"
	"github.com/smap-hcmut/shared-libs/go/log"
)

func main() {
	ctx := context.Background()

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

	postgresDB, err := configPostgre.Connect(ctx, cfg.Postgres)
	if err != nil {
		logger.Fatalf(ctx, "PostgreSQL connect failed: %v", err)
	}
	defer configPostgre.Disconnect(ctx, postgresDB)

	rabbitConn, err := configRabbit.Connect(cfg.RabbitMQ)
	if err != nil {
		logger.Fatalf(ctx, "RabbitMQ connect failed: %v", err)
	}
	defer configRabbit.Disconnect()

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

	schedulerSrv, err := scheduler.New(logger, scheduler.Config{
		DB:        postgresDB,
		AMQPConn:  rabbitConn,
		Scheduler: cfg.Scheduler,
		Discord:   discordClient,
	})
	if err != nil {
		logger.Fatalf(ctx, "Failed to create scheduler: %v", err)
	}

	if err := schedulerSrv.Start(); err != nil {
		logger.Fatalf(ctx, "Failed to start scheduler: %v", err)
	}
}
