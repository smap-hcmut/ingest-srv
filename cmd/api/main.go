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
	configRedis "ingest-srv/config/redis"
	"ingest-srv/internal/httpserver"
	"ingest-srv/pkg/discord"
	"ingest-srv/pkg/encrypter"
	"ingest-srv/pkg/jwt"
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

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	logger.Info(ctx, "Starting Ingest API Service...")

	enc := encrypter.New(cfg.Encrypter.Key)

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

	jwtManager, err := jwt.New(jwt.Config{SecretKey: cfg.JWT.SecretKey})
	if err != nil {
		logger.Error(ctx, "Failed to initialize JWT manager: ", err)
		return
	}

	postgresDB, err := configPostgre.Connect(ctx, cfg.Postgres)
	if err != nil {
		logger.Errorf(ctx, "PostgreSQL connect failed, service continues with degraded readiness: %v", err)
		postgresDB = nil
	} else {
		defer configPostgre.Disconnect(ctx, postgresDB)
		logger.Info(ctx, "PostgreSQL client initialized")
	}

	redisClient, err := configRedis.Connect(ctx, cfg.Redis)
	if err != nil {
		logger.Errorf(ctx, "Redis connect failed, service continues with degraded readiness: %v", err)
		redisClient = nil
	} else {
		defer configRedis.Disconnect()
		logger.Info(ctx, "Redis client initialized")
	}

	minioClient, err := configMinio.Connect(ctx, &cfg.MinIO)
	if err != nil {
		logger.Errorf(ctx, "MinIO connect failed, service continues in degraded mode: %v", err)
		minioClient = nil
	} else {
		defer configMinio.Disconnect()
		logger.Info(ctx, "MinIO client initialized")
	}

	kafkaProducer, err := configKafka.ConnectProducer(cfg.Kafka)
	if err != nil {
		logger.Errorf(ctx, "Kafka producer connect failed, service continues in degraded mode: %v", err)
		kafkaProducer = nil
	} else {
		defer configKafka.DisconnectProducer()
		logger.Info(ctx, "Kafka producer initialized")
	}

	rabbitConn, err := configRabbit.Connect(cfg.RabbitMQ)
	if err != nil {
		logger.Errorf(ctx, "RabbitMQ connect failed, service continues in degraded mode: %v", err)
		rabbitConn = nil
	} else {
		defer configRabbit.Disconnect()
		logger.Info(ctx, "RabbitMQ connection initialized")
	}

	httpSrv, err := httpserver.New(logger, httpserver.Config{
		Logger:      logger,
		Host:        cfg.HTTPServer.Host,
		Port:        cfg.HTTPServer.Port,
		Mode:        cfg.HTTPServer.Mode,
		Environment: cfg.Environment.Name,

		PostgresDB: postgresDB,
		Redis:      redisClient,
		MinIO:      minioClient,
		Kafka:      kafkaProducer,
		RabbitMQ:   rabbitConn,

		Config:       cfg,
		JWTManager:   jwtManager,
		CookieConfig: cfg.Cookie,
		Encrypter:    enc,
		Discord:      discordClient,
	})
	if err != nil {
		logger.Error(ctx, "Failed to initialize HTTP server: ", err)
		return
	}

	if err := httpSrv.Run(); err != nil {
		logger.Error(ctx, "Failed to run API server: ", err)
		return
	}

	logger.Info(ctx, "Ingest API service stopped gracefully")
}
