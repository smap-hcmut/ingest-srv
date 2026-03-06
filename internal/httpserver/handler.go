package httpserver

import (
	"context"

	datasourceHandler "ingest-srv/internal/datasource/delivery/http"
	datasourceRepo "ingest-srv/internal/datasource/repository/postgre"
	datasourceUC "ingest-srv/internal/datasource/usecase"

	"ingest-srv/internal/middleware"
	"ingest-srv/pkg/response"

	"github.com/gin-gonic/gin"
)

func (srv HTTPServer) mapHandlers() error {
	mw := middleware.New(
		srv.l,
		srv.jwtManager,
		srv.cookieConfig,
		srv.cfg.InternalConfig.InternalKey,
	)

	srv.registerMiddlewares(mw)
	srv.registerSystemRoutes()
	srv.registerIngestRoutes()

	dataSRepo := datasourceRepo.New(srv.l, srv.postgresDB)
	dataUC := datasourceUC.New(srv.l, dataSRepo)
	datasourceHandler := datasourceHandler.New(srv.l, dataUC, srv.discord)

	api := srv.gin.Group("/api/v1")
	datasourceHandler.RegisterRoutes(api, mw)

	return nil
}

func (srv HTTPServer) registerMiddlewares(mw middleware.Middleware) {
	srv.gin.Use(middleware.Recovery(srv.l, srv.discord))

	corsConfig := middleware.DefaultCORSConfig(srv.environment)
	srv.gin.Use(middleware.CORS(corsConfig))

	ctx := context.Background()
	if srv.environment == "production" {
		srv.l.Infof(ctx, "CORS mode: production")
	} else {
		srv.l.Infof(ctx, "CORS mode: %s", srv.environment)
	}

	srv.gin.Use(mw.Locale())
}

func (srv HTTPServer) registerSystemRoutes() {
	srv.gin.GET("/health", srv.healthCheck)
	srv.gin.GET("/ready", srv.readyCheck)
	srv.gin.GET("/live", srv.liveCheck)
}

func (srv HTTPServer) registerIngestRoutes() {
	api := srv.gin.Group("/api/v1/ingest")
	api.GET("/ping", func(c *gin.Context) {
		response.OK(c, gin.H{
			"service": "ingest-srv",
			"message": "ingest api boilerplate is running",
		})
	})
}
