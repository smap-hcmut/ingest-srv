package job

import (
	"ingest-srv/config"
	"ingest-srv/internal/execution"
	"ingest-srv/pkg/cron"
	"ingest-srv/pkg/log"
)

type Handler interface {
	DispatchDueTargets()
	Register() []cron.JobInfo
}

type handler struct {
	l    log.Logger
	cfg  config.SchedulerConfig
	uc   execution.UseCase
	cron cron.Cron
}

// New creates a new execution scheduler job handler.
func New(l log.Logger, cfg config.SchedulerConfig, cronJ cron.Cron, uc execution.UseCase) Handler {
	return handler{
		l:    l,
		cfg:  cfg,
		uc:   uc,
		cron: cronJ,
	}
}
