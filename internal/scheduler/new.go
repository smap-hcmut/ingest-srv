package scheduler

import (
	"database/sql"
	"errors"

	"ingest-srv/config"
	"ingest-srv/pkg/cron"
	"ingest-srv/pkg/discord"
	"ingest-srv/pkg/log"
	"ingest-srv/pkg/rabbitmq"
)

type Scheduler struct {
	cron       cron.Cron
	l          log.Logger
	db         *sql.DB
	conn       rabbitmq.IRabbitMQ
	cfg        config.SchedulerConfig
	discordApp discord.IDiscord
}

type Config struct {
	DB        *sql.DB
	AMQPConn  rabbitmq.IRabbitMQ
	Scheduler config.SchedulerConfig
	Discord   discord.IDiscord
}

// New creates scheduler runtime.
func New(l log.Logger, cfg Config) (Scheduler, error) {
	s := Scheduler{
		cron:       cron.New(),
		l:          l,
		db:         cfg.DB,
		conn:       cfg.AMQPConn,
		cfg:        cfg.Scheduler,
		discordApp: cfg.Discord,
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
	}

	for _, dep := range requiredDeps {
		if !dep.ok {
			return errors.New(dep.msg)
		}
	}

	return nil
}
