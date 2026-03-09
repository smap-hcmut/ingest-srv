package http

import (
	"ingest-srv/internal/middleware"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes maps dryrun routes under datasource resources.
func (h *handler) RegisterRoutes(r *gin.RouterGroup, mw middleware.Middleware) {
	sources := r.Group("/datasources")
	sources.Use(mw.Auth())
	{
		sources.POST("/:id/dryrun", h.Trigger)
		sources.GET("/:id/dryrun/latest", h.GetLatest)
		sources.GET("/:id/dryrun/history", h.ListHistory)
	}
}
