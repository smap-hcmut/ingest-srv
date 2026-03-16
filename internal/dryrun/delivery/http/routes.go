package http

import (
	"github.com/gin-gonic/gin"
	"github.com/smap-hcmut/shared-libs/go/middleware"
)

// RegisterRoutes maps dryrun routes under datasource resources.
func (h *handler) RegisterRoutes(r *gin.RouterGroup, mw *middleware.Middleware) {
	sources := r.Group("/datasources")
	sources.Use(mw.Auth())
	{
		sources.POST("/:id/dryrun", h.Trigger)
		sources.GET("/:id/dryrun/latest", h.GetLatest)
		sources.GET("/:id/dryrun/history", h.ListHistory)
	}
}
