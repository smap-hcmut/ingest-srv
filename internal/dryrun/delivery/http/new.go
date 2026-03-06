package http

import (
	"ingest-srv/internal/dryrun"
	"ingest-srv/internal/middleware"
	"ingest-srv/pkg/discord"
	"ingest-srv/pkg/log"

	"github.com/gin-gonic/gin"
)

// Handler defines the HTTP handler interface for dryrun operations.
type Handler interface {
	RegisterRoutes(r *gin.RouterGroup, mw middleware.Middleware)
}

type handler struct {
	l       log.Logger
	uc      dryrun.UseCase
	discord discord.IDiscord
}

// New creates a new dryrun HTTP handler.
func New(l log.Logger, uc dryrun.UseCase, discord discord.IDiscord) Handler {
	return &handler{l: l, uc: uc, discord: discord}
}
