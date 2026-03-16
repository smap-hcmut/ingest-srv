package http

import (
	"ingest-srv/internal/execution"

	"github.com/gin-gonic/gin"
	"github.com/smap-hcmut/shared-libs/go/discord"
	"github.com/smap-hcmut/shared-libs/go/log"
	"github.com/smap-hcmut/shared-libs/go/middleware"
)

// Handler exposes internal execution endpoints.
type Handler interface {
	RegisterInternalRoutes(r *gin.RouterGroup, mw *middleware.Middleware)
}

type handler struct {
	l       log.Logger
	uc      execution.UseCase
	discord discord.IDiscord
}

// New creates a new execution HTTP handler.
func New(l log.Logger, uc execution.UseCase, discord discord.IDiscord) Handler {
	return &handler{
		l:       l,
		uc:      uc,
		discord: discord,
	}
}
