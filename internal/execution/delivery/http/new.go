package http

import (
	"ingest-srv/internal/execution"
	"ingest-srv/internal/middleware"
	"ingest-srv/pkg/discord"
	"ingest-srv/pkg/log"

	"github.com/gin-gonic/gin"
)

// Handler exposes internal execution endpoints.
type Handler interface {
	RegisterInternalRoutes(r *gin.RouterGroup, mw middleware.Middleware)
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
