package httpserver

import (
	"database/sql"
	"errors"

	"github.com/gin-gonic/gin"

	"ingest-srv/config"
	"ingest-srv/pkg/discord"
	"ingest-srv/pkg/encrypter"
	"ingest-srv/pkg/jwt"
	"ingest-srv/pkg/kafka"
	"ingest-srv/pkg/log"
	"ingest-srv/pkg/minio"
	"ingest-srv/pkg/rabbitmq"
	"ingest-srv/pkg/redis"
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
	jwtManager   jwt.IManager
	cookieConfig config.CookieConfig
	encrypter    encrypter.Encrypter
	discord      discord.IDiscord
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
	JWTManager   jwt.IManager
	CookieConfig config.CookieConfig
	Encrypter    encrypter.Encrypter
	Discord      discord.IDiscord
}

// New creates a new HTTP server.
func New(logger log.Logger, cfg Config) (*HTTPServer, error) {
	gin.SetMode(cfg.Mode)

	srv := &HTTPServer{
		gin:         gin.Default(),
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
		jwtManager:   cfg.JWTManager,
		cookieConfig: cfg.CookieConfig,
		encrypter:    cfg.Encrypter,
		discord:      cfg.Discord,
	}

	if err := srv.validate(); err != nil {
		return nil, err
	}

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
	if srv.jwtManager == nil {
		return errors.New("jwtManager is required")
	}
	if srv.encrypter == nil {
		return errors.New("encrypter is required")
	}
	return nil
}
