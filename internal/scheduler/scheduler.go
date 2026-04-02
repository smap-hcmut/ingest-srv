package scheduler

import (
	"context"
	"fmt"
	executionJob "ingest-srv/internal/execution/delivery/job"
	executionProducer "ingest-srv/internal/execution/delivery/rabbitmq/producer"
	executionRepo "ingest-srv/internal/execution/repository/postgre"
	executionUC "ingest-srv/internal/execution/usecase"
	projectsrv "ingest-srv/pkg/microservice/project"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"

	"github.com/smap-hcmut/shared-libs/go/cron"
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
	execRepo := executionRepo.New(s.l, s.db)
	execProducer := executionProducer.New(s.l, s.conn)
	projectSrv := projectsrv.New(s.l, s.microservice.Project.BaseURL, s.microservice.Project.TimeoutMS, s.internalKey)
	if err := execProducer.Run(); err != nil {
		return err
	}

	execUC := executionUC.New(s.l, execRepo, nil, execProducer, nil, projectSrv)
	jobHandler := executionJob.New(s.l, s.cfg, s.cron, execUC)

	// Register jobs using shared-libs cron
	jobInfos := jobHandler.Register()
	for _, jobInfo := range jobInfos {
		if err := s.cron.AddJob(jobInfo); err != nil {
			return fmt.Errorf("failed to add job %s: %w", jobInfo.Name, err)
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
