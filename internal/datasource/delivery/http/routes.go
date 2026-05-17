package http

import (
	"github.com/gin-gonic/gin"
	"github.com/smap-hcmut/shared-libs/go/middleware"
)

// RegisterRoutes maps data source routes to the given router group.
func (h *handler) RegisterRoutes(r *gin.RouterGroup, mw *middleware.Middleware) {
	sources := r.Group("/datasources")
	sources.Use(mw.Auth())
	{
		sources.POST("", mw.AdminOnly(), h.Create)
		sources.GET("", h.List)
		sources.GET("/:id", h.Detail)
		sources.PUT("/:id", mw.AdminOnly(), h.Update)
		sources.POST("/:id/activate", mw.AdminOnly(), h.ActivateDataSource)
		sources.POST("/:id/pause", mw.AdminOnly(), h.PauseDataSource)
		sources.POST("/:id/resume", mw.AdminOnly(), h.ResumeDataSource)
		sources.POST("/:id/archive", mw.AdminOnly(), h.Archive)
		sources.DELETE("/:id", mw.AdminOnly(), h.Delete)

		// CrawlTarget sub-resource routes.
		targets := sources.Group("/:id/targets")
		{
			targets.POST("/keywords", mw.AdminOnly(), h.CreateKeywordTarget)
			targets.POST("/profiles", mw.AdminOnly(), h.CreateProfileTarget)
			targets.POST("/posts", mw.AdminOnly(), h.CreatePostTarget)
			targets.GET("", h.ListTargets)
			targets.GET("/:target_id", h.DetailTarget)
			targets.PUT("/:target_id", mw.AdminOnly(), h.UpdateTarget)
			targets.POST("/:target_id/activate", mw.AdminOnly(), h.ActivateTarget)
			targets.POST("/:target_id/deactivate", mw.AdminOnly(), h.DeactivateTarget)
			targets.DELETE("/:target_id", mw.AdminOnly(), h.DeleteTarget)
		}
	}
}

// RegisterInternalRoutes maps internal datasource routes.
func (h *handler) RegisterInternalRoutes(r *gin.RouterGroup, mw *middleware.Middleware) {
	sources := r.Group("/datasources")
	sources.Use(mw.InternalAuth())
	{
		sources.PUT("/:id/crawl-mode", h.UpdateCrawlMode)
	}

	projects := r.Group("/projects")
	projects.Use(mw.InternalAuth())
	{
		projects.GET("/:project_id/activation-readiness", h.GetActivationReadiness)
		projects.POST("/:project_id/activate", h.Activate)
		projects.POST("/:project_id/pause", h.Pause)
		projects.POST("/:project_id/resume", h.Resume)
		projects.POST("/:project_id/crawl-mode", h.UpdateProjectCrawlMode)
	}
}
