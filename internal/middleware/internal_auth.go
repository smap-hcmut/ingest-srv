package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/smap-hcmut/shared-libs/go/auth"
)

// InternalAuth validates the internal key from X-Internal-Key header
func (m Middleware) InternalAuth() gin.HandlerFunc {
	return auth.InternalAuth(auth.InternalAuthConfig{ExpectedKey: m.InternalKey})
}
