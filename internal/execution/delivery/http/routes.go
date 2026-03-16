package http

import (
	"github.com/gin-gonic/gin"
	"github.com/smap-hcmut/shared-libs/go/middleware"
)

// RegisterInternalRoutes maps internal execution routes.
func (h *handler) RegisterInternalRoutes(r *gin.RouterGroup, mw *middleware.Middleware) {
	sources := r.Group("/datasources")
	sources.Use(mw.InternalAuth())
	{
		sources.POST("/:id/targets/:target_id/dispatch", h.DispatchTarget)
	}
}
