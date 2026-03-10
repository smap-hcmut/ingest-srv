package http

import (
	"ingest-srv/internal/middleware"

	"github.com/gin-gonic/gin"
)

// RegisterInternalRoutes maps internal execution routes.
func (h *handler) RegisterInternalRoutes(r *gin.RouterGroup, mw middleware.Middleware) {
	sources := r.Group("/datasources")
	sources.Use(mw.InternalAuth())
	{
		sources.POST("/:id/targets/:target_id/dispatch", h.DispatchTarget)
	}
}
