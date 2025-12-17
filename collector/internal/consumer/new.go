package consumer

import (
	"errors"

	"smap-collector/config"
	"smap-collector/internal/dispatcher"
	"smap-collector/internal/state"
	"smap-collector/pkg/discord"
	pkgLog "smap-collector/pkg/log"
	"smap-collector/pkg/rabbitmq"
	pkgRedis "smap-collector/pkg/redis"
)

type Server struct {
	l    pkgLog.Logger
	conn *rabbitmq.Connection
	cfg  Config
	disc *discord.DiscordWebhook
}

// Config contains all dependencies for the consumer server.
type Config struct {
	Logger            pkgLog.Logger
	AMQPConn          *rabbitmq.Connection
	Discord           *discord.DiscordWebhook
	DispatcherOptions dispatcher.Options
	ProjectConfig     config.ProjectConfig
	RedisClient       pkgRedis.Client
	StateOptions      state.Options
	CrawlLimitsConfig config.CrawlLimitsConfig
}

func New(cfg Config) (*Server, error) {
	if cfg.Logger == nil {
		return nil, errors.New("logger is required")
	}
	if cfg.AMQPConn == nil {
		return nil, errors.New("amqp connection is required")
	}

	return &Server{
		l:    cfg.Logger,
		conn: cfg.AMQPConn,
		cfg:  cfg,
		disc: cfg.Discord,
	}, nil
}
