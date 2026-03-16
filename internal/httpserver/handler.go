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
	"ingest-srv/internal/model"

	"github.com/gin-gonic/gin"
	"github.com/smap-hcmut/shared-libs/go/auth"
	"github.com/smap-hcmut/shared-libs/go/middleware"
	"github.com/smap-hcmut/shared-libs/go/response"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func (srv HTTPServer) mapHandlers() error {
	mw := middleware.New(middleware.Config{
		JWTManager:       auth.NewManager(srv.cfg.JWT.SecretKey),
		CookieName:       srv.cookieConfig.Name,
		ProductionDomain: srv.cookieConfig.Domain,
		InternalKey:      srv.cfg.InternalConfig.InternalKey,
	})

	srv.registerMiddlewares()
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

	apiV1 := srv.gin.Group(model.APIV1Prefix)
	datasourceHTTP.RegisterRoutes(apiV1, mw)
	dryrunHTTP.RegisterRoutes(apiV1, mw)
	ingestAPI := apiV1.Group("/ingest")
	datasourceHTTP.RegisterInternalRoutes(ingestAPI, mw)
	execHTTP.RegisterInternalRoutes(ingestAPI, mw)

	return nil
}

func (srv HTTPServer) registerMiddlewares() {
	srv.gin.Use(middleware.Tracing())
	srv.gin.Use(middleware.Recovery(srv.l, srv.discord))

	corsConfig := middleware.DefaultCORSConfig(srv.environment)
	srv.gin.Use(middleware.CORS(corsConfig))

	ctx := context.Background()
	if srv.environment == "production" {
		srv.l.Infof(ctx, "CORS mode: production")
	} else {
		srv.l.Infof(ctx, "CORS mode: %s", srv.environment)
	}

	srv.gin.Use(middleware.Locale())
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
	api := srv.gin.Group(model.APIV1Prefix + "/ingest")
	api.GET("/ping", func(c *gin.Context) {
		response.OK(c, gin.H{
			"service": "ingest-srv",
			"message": "ingest api boilerplate is running",
		})
	})
}
