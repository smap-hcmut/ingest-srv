package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/smap-hcmut/shared-libs/go/response"
	"github.com/smap-hcmut/shared-libs/go/scope"
)

func (m Middleware) Auth() gin.HandlerFunc {
	return func(c *gin.Context) {
		var tokenString string
		var err error

		// Priority 1: Try to read token from Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			// Support both "Bearer <token>" and plain token
			if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
				tokenString = authHeader[7:]
			} else {
				tokenString = authHeader
			}
		}

		// Priority 2: If no token in header (or it's just "Bearer "), try cookie
		if tokenString == "" || tokenString == "Bearer " {
			tokenString, err = c.Cookie(m.cookieConfig.Name)
			if err != nil || tokenString == "" {
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
