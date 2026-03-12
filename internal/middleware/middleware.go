package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/smap-hcmut/shared-libs/go/response"
	"github.com/smap-hcmut/shared-libs/go/scope"
)

func (m Middleware) Auth() gin.HandlerFunc {
	return func(c *gin.Context) {

		// Cookie-first authentication strategy
		// First, attempt to read token from cookie (preferred method)
		tokenString, err := c.Cookie(m.cookieConfig.Name)

		// Fallback to Authorization header for backward compatibility
		if err != nil || tokenString == "" {
			authHeader := c.GetHeader("Authorization")
			if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
				tokenString = authHeader[7:]
			} else {
				response.Unauthorized(c)
				c.Abort()
				return
			}
		}

		payload, err := m.jwtManager.Verify(tokenString)
		if err != nil {
			response.Unauthorized(c)
			c.Abort()
			return
		}

		ctx := c.Request.Context()
		ctx = scope.SetPayloadToContext(ctx, payload)
		sc := scope.NewScope(payload)
		ctx = scope.SetScopeToContext(ctx, sc)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}

func (m Middleware) InternalAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Read internal API key from X-Internal-Key header
		// This is used for service-to-service communication
		tokenString := c.GetHeader("X-Internal-Key")

		if tokenString == "" {
			response.Unauthorized(c)
			c.Abort()
			return
		}

		if tokenString != m.InternalKey {
			response.Unauthorized(c)
			c.Abort()
			return
		}

		c.Next()
	}
}
