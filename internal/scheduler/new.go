package scheduler

import (
	"fmt"

	"ingest-srv/config"
	"ingest-srv/pkg/cron"
	"ingest-srv/pkg/discord"
	"ingest-srv/pkg/log"
)

// Scheduler represents ingest scheduler runtime.
type Scheduler struct {
	cron    cron.Cron
	l       log.Logger
	cfg     config.SchedulerConfig
	discord discord.IDiscord
}

// Config is dependency bag for scheduler.
type Config struct {
	Logger  log.Logger
	Config  config.SchedulerConfig
	Discord discord.IDiscord
}

// New creates scheduler instance.
func New(cfg Config) (*Scheduler, error) {
	s := &Scheduler{
		cron:    cron.New(),
		l:       cfg.Logger,
		cfg:     cfg.Config,
		discord: cfg.Discord,
	}

	if err := s.validate(); err != nil {
		return nil, err
	}

	return s, nil
}

func (s *Scheduler) validate() error {
	if s.l == nil {
		return fmt.Errorf("logger is required")
	}
	if s.cfg.HeartbeatCron == "" {
		return fmt.Errorf("scheduler heartbeat cron is required")
	}
	if s.cfg.Timezone == "" {
		return fmt.Errorf("scheduler timezone is required")
	}
	return nil
}
