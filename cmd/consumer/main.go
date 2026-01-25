package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"smap-collector/config"
	"smap-collector/internal/consumer"
	"smap-collector/internal/state"
	"smap-collector/pkg/discord"
	pkgLog "smap-collector/pkg/log"
	"smap-collector/pkg/rabbitmq"
	pkgRedis "smap-collector/pkg/redis"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	l := pkgLog.Init(pkgLog.ZapConfig{
		Level:        cfg.Logger.Level,
		Mode:         cfg.Logger.Mode,
		Encoding:     cfg.Logger.Encoding,
		ColorEnabled: cfg.Logger.ColorEnabled,
	})

	// Initialize Discord webhook for error reporting
	discordWebhook, err := discord.NewDiscordWebhook(cfg.Discord.ReportBugID, cfg.Discord.ReportBugToken)
	if err != nil {
		l.Warnf(ctx, "failed to initialize Discord webhook: %v", err)
	}

	// Connect to RabbitMQ (fail fast)
	conn, err := rabbitmq.Dial(cfg.RabbitMQConfig.URL, true)
	if err != nil {
		l.Fatalf(ctx, "failed to connect to RabbitMQ: %v", err)
	}
	defer conn.Close()

	// Initialize Redis client for state management (fail fast)
	redisOpts := pkgRedis.NewClientOptions().SetOptions(cfg.Redis).SetDB(cfg.Redis.StateDB)
	redisClient, err := pkgRedis.Connect(redisOpts)
	if err != nil {
		l.Fatalf(ctx, "failed to connect to Redis: %v", err)
	}
	defer redisClient.Disconnect()

	if err := redisClient.Ping(ctx); err != nil {
		l.Fatalf(ctx, "Redis ping failed: %v", err)
	}
	l.Infof(ctx, "Redis state client connected: db=%d", cfg.Redis.StateDB)

	// Create consumer server with initialized dependencies
	srv, err := consumer.New(consumer.Config{
		Logger:            l,
		AMQPConn:          conn,
		Discord:           discordWebhook,
		ProjectConfig:     cfg.Project,
		RedisClient:       redisClient,
		StateOptions:      state.Options{TTL: state.DefaultTTL},
		CrawlLimitsConfig: cfg.CrawlLimits,
	})
	if err != nil {
		l.Fatalf(ctx, "failed to init consumer: %v", err)
	}
	defer srv.Close()

	l.Info(ctx, "Starting SMAP Dispatcher Service...")

	if err := srv.Run(ctx); err != nil {
		l.Fatalf(ctx, "consumer stopped with error: %v", err)
	}
}
