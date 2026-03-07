package scheduler

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"

	executionJob "ingest-srv/internal/execution/delivery/job"
	executionProducer "ingest-srv/internal/execution/delivery/rabbitmq/producer"
	executionRepo "ingest-srv/internal/execution/repository/postgre"
	executionUC "ingest-srv/internal/execution/usecase"
	"ingest-srv/pkg/cron"
)

func (s Scheduler) Start() error {
	ctx := context.Background()

	s.l.Info(ctx, "Starting ingest scheduler")

	if err := s.registerJobs(); err != nil {
		return err
	}

	go func() {
		s.l.Info(ctx, "Starting scheduler cron")
		s.cron.Start()
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	s.l.Info(ctx, "Stopping ingest scheduler")
	s.cron.Stop()

	return nil
}

func (s Scheduler) registerJobs() error {
	s.cron.SetFuncWrapper(func(f cron.HandleFunc) {
		s.jobWrapper(f)
	})

	execRepo := executionRepo.New(s.l, s.db)
	execProducer := executionProducer.New(s.l, s.conn)
	if err := execProducer.Run(); err != nil {
		return err
	}

	execUC := executionUC.New(s.l, execRepo, nil, execProducer)

	jobHandlers := []interface {
		Register() []cron.JobInfo
	}{
		executionJob.New(s.l, s.cfg, s.cron, execUC),
	}

	for _, jobHandler := range jobHandlers {
		infos := jobHandler.Register()
		for _, info := range infos {
			if err := s.cron.AddJob(info); err != nil {
				return err
			}
		}
	}

	return nil
}

func (s Scheduler) jobWrapper(f cron.HandleFunc) {
	defer func() {
		if err := recover(); err != nil {
			ctx := context.Background()
			errMsg := fmt.Sprintf("scheduler panic: %v\n%s", err, string(debug.Stack()))
			s.l.Errorf(ctx, errMsg)
			if s.discordApp != nil {
				_ = s.discordApp.ReportBug(ctx, errMsg)
			}
		}
	}()

	f()
}
