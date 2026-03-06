package scheduler

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"
	"time"

	"ingest-srv/pkg/cron"
)

// Start starts scheduler and blocks until shutdown signal.
func (s *Scheduler) Start() error {
	ctx := context.Background()
	s.l.Info(ctx, "Starting ingest scheduler...")

	if err := s.registerJobs(); err != nil {
		return err
	}

	go func() {
		s.l.Info(ctx, "Scheduler cron loop started")
		s.cron.Start()
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	s.l.Info(ctx, "Stopping scheduler...")
	s.cron.Stop()
	s.l.Info(ctx, "Scheduler stopped")
	return nil
}

func (s *Scheduler) registerJobs() error {
	s.cron.SetFuncWrapper(func(f cron.HandleFunc) {
		s.jobWrapper(f)
	})

	if err := s.cron.AddJob(cron.JobInfo{
		CronTime: s.cfg.HeartbeatCron,
		Handler: func() {
			loc, err := time.LoadLocation(s.cfg.Timezone)
			now := time.Now()
			if err == nil {
				now = now.In(loc)
			}
			s.l.Infof(context.Background(), "[scheduler-heartbeat] tick at %s", now.Format(time.RFC3339))
		},
	}); err != nil {
		return fmt.Errorf("failed to register heartbeat cron: %w", err)
	}

	return nil
}

func (s *Scheduler) jobWrapper(f cron.HandleFunc) {
	defer func() {
		if r := recover(); r != nil {
			ctx := context.Background()
			errMsg := fmt.Sprintf("scheduler panic: %v\n%s", r, string(debug.Stack()))
			s.l.Errorf(ctx, errMsg)
			if s.discord != nil {
				_ = s.discord.ReportBug(ctx, errMsg)
			}
		}
	}()

	f()
}
