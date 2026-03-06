package http

import (
	"ingest-srv/internal/middleware"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes maps data source routes to the given router group.
func (h *handler) RegisterRoutes(r *gin.RouterGroup, mw middleware.Middleware) {
	sources := r.Group("/datasources")
	sources.Use(mw.Auth())
	{
		sources.POST("", h.Create)
		sources.GET("", h.List)
		sources.GET("/:id", h.Detail)
		sources.PUT("/:id", h.Update)
		sources.DELETE("/:id", h.Archive)

		// CrawlTarget sub-resource routes.
		targets := sources.Group("/:id/targets")
		{
			targets.POST("", h.CreateTarget)
			targets.GET("", h.ListTargets)
			targets.PUT("/:target_id", h.UpdateTarget)
			targets.DELETE("/:target_id", h.DeleteTarget)
		}
	}
}
