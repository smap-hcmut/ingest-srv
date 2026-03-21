package httpserver

import (
	"database/sql"
	"errors"
	"ingest-srv/config"

	"github.com/gin-gonic/gin"
	"github.com/smap-hcmut/shared-libs/go/discord"
	"github.com/smap-hcmut/shared-libs/go/encrypter"
	"github.com/smap-hcmut/shared-libs/go/kafka"
	"github.com/smap-hcmut/shared-libs/go/log"
	"github.com/smap-hcmut/shared-libs/go/middleware"
	"github.com/smap-hcmut/shared-libs/go/minio"
	"github.com/smap-hcmut/shared-libs/go/rabbitmq"
	"github.com/smap-hcmut/shared-libs/go/redis"
)

// HTTPServer represents ingest HTTP API server.
type HTTPServer struct {
	gin         *gin.Engine
	l           log.Logger
	host        string
	port        int
	mode        string
	environment string

	postgresDB *sql.DB
	redis      redis.IRedis
	minio      minio.MinIO
	kafka      kafka.IProducer
	rabbitmq   rabbitmq.IRabbitMQ

	cfg          *config.Config
	cookieConfig config.CookieConfig
	encrypter    encrypter.Encrypter
	discord      discord.IDiscord
	microservice Microservice
}

// Config is dependency bag for HTTPServer.
type Config struct {
	Logger      log.Logger
	Host        string
	Port        int
	Mode        string
	Environment string

	PostgresDB *sql.DB
	Redis      redis.IRedis
	MinIO      minio.MinIO
	Kafka      kafka.IProducer
	RabbitMQ   rabbitmq.IRabbitMQ

	Config       *config.Config
	CookieConfig config.CookieConfig
	Encrypter    encrypter.Encrypter
	Discord      discord.IDiscord
	Microservice Microservice
}

type Microservice struct {
	Project ProjectService
}

type ProjectService struct {
	BaseURL   string
	TimeoutMS int
}

// New creates a new HTTP server.
func New(logger log.Logger, cfg Config) (*HTTPServer, error) {
	gin.SetMode(cfg.Mode)

	srv := &HTTPServer{
		gin:         gin.New(),
		l:           logger,
		host:        cfg.Host,
		port:        cfg.Port,
		mode:        cfg.Mode,
		environment: cfg.Environment,

		postgresDB: cfg.PostgresDB,
		redis:      cfg.Redis,
		minio:      cfg.MinIO,
		kafka:      cfg.Kafka,
		rabbitmq:   cfg.RabbitMQ,

		cfg:          cfg.Config,
		cookieConfig: cfg.CookieConfig,
		encrypter:    cfg.Encrypter,
		discord:      cfg.Discord,
		microservice: cfg.Microservice,
	}

	if err := srv.validate(); err != nil {
		return nil, err
	}

	// Add middlewares
	srv.gin.Use(middleware.Logger(srv.l, srv.environment))
	srv.gin.Use(gin.Recovery())

	return srv, nil
}

func (srv HTTPServer) validate() error {
	if srv.l == nil {
		return errors.New("logger is required")
	}
	if srv.mode == "" {
		return errors.New("mode is required")
	}
	if srv.port == 0 {
		return errors.New("port is required")
	}
	if srv.cfg == nil {
		return errors.New("config is required")
	}
	if srv.encrypter == nil {
		return errors.New("encrypter is required")
	}
	if srv.microservice.Project.BaseURL == "" {
		return errors.New("microservice.project.base_url is required")
	}
	if srv.microservice.Project.TimeoutMS <= 0 {
		return errors.New("microservice.project.timeout_ms must be greater than 0")
	}
	return nil
}
