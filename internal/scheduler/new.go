package scheduler

import (
	"database/sql"
	"errors"

	"ingest-srv/config"

	"github.com/smap-hcmut/shared-libs/go/cron"
	"github.com/smap-hcmut/shared-libs/go/discord"
	"github.com/smap-hcmut/shared-libs/go/log"
	"github.com/smap-hcmut/shared-libs/go/rabbitmq"
)

type Scheduler struct {
	cron         cron.Cron
	l            log.Logger
	db           *sql.DB
	conn         rabbitmq.IRabbitMQ
	cfg          config.SchedulerConfig
	microservice config.MicroserviceConfig
	internalKey  string
	discordApp   discord.IDiscord
}

type Config struct {
	DB           *sql.DB
	AMQPConn     rabbitmq.IRabbitMQ
	Scheduler    config.SchedulerConfig
	Microservice config.MicroserviceConfig
	InternalKey  string
	Discord      discord.IDiscord
}

// New creates scheduler runtime.
func New(l log.Logger, cfg Config) (Scheduler, error) {
	cronScheduler := cron.New()

	s := Scheduler{
		cron:         cronScheduler,
		l:            l,
		db:           cfg.DB,
		conn:         cfg.AMQPConn,
		cfg:          cfg.Scheduler,
		microservice: cfg.Microservice,
		internalKey:  cfg.InternalKey,
		discordApp:   cfg.Discord,
	}
	if err := s.validate(); err != nil {
		return Scheduler{}, err
	}
	return s, nil
}

func (s Scheduler) validate() error {
	requiredDeps := []struct {
		ok  bool
		msg string
	}{
		{s.l != nil, "logger is required"},
		{s.db != nil, "database is required"},
		{s.conn != nil, "amqp connection is required"},
		{s.cfg.HeartbeatCron != "", "scheduler heartbeat cron is required"},
		{s.cfg.Timezone != "", "scheduler timezone is required"},
		{s.cfg.HeartbeatLimit > 0, "scheduler heartbeat limit must be greater than 0"},
		{s.microservice.Project.BaseURL != "", "microservice.project.base_url is required"},
		{s.microservice.Project.TimeoutMS > 0, "microservice.project.timeout_ms must be greater than 0"},
	}

	for _, dep := range requiredDeps {
		if !dep.ok {
			return errors.New(dep.msg)
		}
	}

	return nil
}
