package httpserver

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// Run starts the HTTP server and blocks until a shutdown signal is received.
func (srv HTTPServer) Run() error {
	if err := srv.mapHandlers(); err != nil {
		srv.l.Fatalf(context.Background(), "Failed to map handlers: %v", err)
		return err
	}

	addr := fmt.Sprintf("%s:%d", srv.host, srv.port)
	server := &http.Server{
		Addr:    addr,
		Handler: srv.gin,
	}

	serveErr := make(chan error, 1)
	go func() {
		srv.l.Infof(context.Background(), "API server listening on %s", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serveErr <- err
		}
	}()

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serveErr:
		return err
	case sig := <-ch:
		srv.l.Infof(context.Background(), "Received signal %v, shutting down gracefully", sig)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		srv.l.Errorf(context.Background(), "Server shutdown error: %v", err)
		return err
	}

	srv.l.Info(context.Background(), "API server stopped")
	return nil
}
