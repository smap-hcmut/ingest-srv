package httpserver

import (
	"context"

	datasourceHandler "ingest-srv/internal/datasource/delivery/http"
	datasourceRepo "ingest-srv/internal/datasource/repository/postgre"
	datasourceUC "ingest-srv/internal/datasource/usecase"
	dryrunHandler "ingest-srv/internal/dryrun/delivery/http"
	dryrunRepo "ingest-srv/internal/dryrun/repository/postgre"
	dryrunUC "ingest-srv/internal/dryrun/usecase"
	executionHandler "ingest-srv/internal/execution/delivery/http"
	executionProducer "ingest-srv/internal/execution/delivery/rabbitmq/producer"
	executionRepo "ingest-srv/internal/execution/repository/postgre"
	executionUC "ingest-srv/internal/execution/usecase"
	"ingest-srv/internal/middleware"
	sharedmw "github.com/smap-hcmut/shared-libs/go/middleware"

	"github.com/gin-gonic/gin"
	"github.com/smap-hcmut/shared-libs/go/response"
	"github.com/smap-hcmut/shared-libs/go/auth"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func (srv HTTPServer) mapHandlers() error {
	scopeManager := auth.NewManager(srv.cfg.JWT.SecretKey)
	mw := middleware.New(
		srv.l,
		scopeManager,
		srv.cookieConfig,
		srv.cfg.InternalConfig.InternalKey,
	)

	srv.registerMiddlewares(mw)
	srv.registerSystemRoutes()
	srv.registerIngestRoutes()

	dataSRepo := datasourceRepo.New(srv.l, srv.postgresDB)
	dataUC := datasourceUC.New(srv.l, dataSRepo)
	datasourceHTTP := datasourceHandler.New(srv.l, dataUC, srv.discord)
	dryrunResultRepo := dryrunRepo.New(srv.l, srv.postgresDB)
	dryrunUseCase := dryrunUC.New(srv.l, dryrunResultRepo, dataSRepo)
	dryrunHTTP := dryrunHandler.New(srv.l, dryrunUseCase, srv.discord)
	execRepo := executionRepo.New(srv.l, srv.postgresDB)
	execProducer := executionProducer.New(srv.l, srv.rabbitmq)
	if err := execProducer.Run(); err != nil {
		return err
	}
	execUseCase := executionUC.New(srv.l, execRepo, srv.minio, execProducer, nil)
	execHTTP := executionHandler.New(srv.l, execUseCase, srv.discord)

	api := srv.gin.Group("/api/v1")
	datasourceHTTP.RegisterRoutes(api, mw)
	dryrunHTTP.RegisterRoutes(api, mw)
	ingestAPI := srv.gin.Group("/api/v1/ingest")
	datasourceHTTP.RegisterInternalRoutes(ingestAPI, mw)
	execHTTP.RegisterInternalRoutes(ingestAPI, mw)

	return nil
}

func (srv HTTPServer) registerMiddlewares(mw middleware.Middleware) {
	srv.gin.Use(sharedmw.Tracing())
	srv.gin.Use(sharedmw.Recovery(srv.l, srv.discord))

	corsConfig := sharedmw.DefaultCORSConfig(srv.environment)
	srv.gin.Use(sharedmw.CORS(corsConfig))

	ctx := context.Background()
	if srv.environment == "production" {
		srv.l.Infof(ctx, "CORS mode: production")
	} else {
		srv.l.Infof(ctx, "CORS mode: %s", srv.environment)
	}

	srv.gin.Use(sharedmw.Locale())
}

func (srv HTTPServer) registerSystemRoutes() {
	srv.gin.GET("/health", srv.healthCheck)
	srv.gin.GET("/ready", srv.readyCheck)
	srv.gin.GET("/live", srv.liveCheck)

	// Swagger UI and docs
	srv.gin.GET("/swagger/*any", ginSwagger.WrapHandler(
		swaggerFiles.Handler,
		ginSwagger.URL("doc.json"),
		ginSwagger.DefaultModelsExpandDepth(-1),
	))
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
