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
	_ "ingest-srv/docs" // Swagger docs - blank import to trigger init()
	"ingest-srv/internal/httpserver"

	"github.com/smap-hcmut/shared-libs/go/discord"
	"github.com/smap-hcmut/shared-libs/go/encrypter"
	"github.com/smap-hcmut/shared-libs/go/log"
	"github.com/smap-hcmut/shared-libs/go/redis"
	_ "github.com/smap-hcmut/shared-libs/go/response" // For swagger type definitions
)

// @title       SMAP Ingest Service API
// @description SMAP Ingest Service API documentation.
// @version     1
// @host        localhost:8080
// @schemes     http https
// @BasePath    /api/v1
//
// @securityDefinitions.apikey CookieAuth
// @in cookie
// @name smap_auth_token
// @description Authentication token stored in HttpOnly cookie.
//
// @securityDefinitions.apikey Bearer
// @in header
// @name Authorization
// @description Bearer token. Format: "Bearer {token}"
//
// @securityDefinitions.apikey InternalKey
// @in header
// @name X-Internal-Key
// @description Internal service-to-service authentication key.
func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Println("Failed to load config:", err)
		return
	}

	logger := log.NewZapLogger(log.ZapConfig{
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

	postgresDB, err := configPostgre.Connect(ctx, cfg.Postgres)
	if err != nil {
		logger.Errorf(ctx, "PostgreSQL connect failed, service continues with degraded readiness: %v", err)
		postgresDB = nil
	} else {
		defer configPostgre.Disconnect(ctx, postgresDB)
		logger.Info(ctx, "PostgreSQL client initialized")
	}

	redisClient, err := redis.New(redis.RedisConfig{
		Host:     cfg.Redis.Host,
		Port:     cfg.Redis.Port,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	if err != nil {
		logger.Errorf(ctx, "Redis connect failed, service continues with degraded readiness: %v", err)
		redisClient = nil
	} else {
		defer redisClient.Close()
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
		CookieConfig: cfg.Cookie,
		Encrypter:    enc,
		Discord:      discordClient,
		Microservice: httpserver.Microservice{
			Project: httpserver.ProjectService{
				BaseURL:   cfg.Microservice.Project.BaseURL,
				TimeoutMS: cfg.Microservice.Project.TimeoutMS,
			},
		},
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
