package middleware

import (
	"ingest-srv/config"

	"github.com/smap-hcmut/shared-libs/go/log"
	"github.com/smap-hcmut/shared-libs/go/scope"
)

type Middleware struct {
	l            log.Logger
	jwtManager   scope.Manager
	cookieConfig config.CookieConfig
	InternalKey  string
}

func New(l log.Logger, jwtManager scope.Manager, cookieConfig config.CookieConfig, internalKey string) Middleware {
	return Middleware{
		l:            l,
		jwtManager:   jwtManager,
		cookieConfig: cookieConfig,
		InternalKey:  internalKey,
	}
}
