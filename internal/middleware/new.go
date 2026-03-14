package middleware

import (
	"os"
	"ingest-srv/config"

	"github.com/gin-gonic/gin"
	"github.com/smap-hcmut/shared-libs/go/auth"
	"github.com/smap-hcmut/shared-libs/go/log"
)

// Middleware wraps shared-libs auth.Middleware for backward compatibility
type Middleware struct {
	authMiddleware *auth.Middleware
	l              log.Logger
	cookieConfig   config.CookieConfig
	InternalKey    string
}

// New creates a new middleware using shared-libs auth.Middleware
func New(l log.Logger, jwtManager auth.Manager, cookieConfig config.CookieConfig, internalKey string) Middleware {
	authMiddleware := auth.NewMiddleware(auth.MiddlewareConfig{
		Manager:                 jwtManager,
		CookieName:              cookieConfig.Name,
		AllowBearerInProduction: os.Getenv("ENVIRONMENT_NAME") != "production",
		ProductionDomain:        cookieConfig.Domain,
	})

	return Middleware{
		authMiddleware: authMiddleware,
		l:              l,
		cookieConfig:   cookieConfig,
		InternalKey:    internalKey,
	}
}

// Auth returns the Gin authentication middleware
func (m Middleware) Auth() gin.HandlerFunc {
	return m.authMiddleware.Authenticate()
}
