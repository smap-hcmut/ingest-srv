package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/smap-hcmut/shared-libs/go/auth"
	"github.com/smap-hcmut/shared-libs/go/response"
)

// AdminOnly is a middleware that checks if the user has admin role
// This middleware should be used after Auth() middleware
func (m Middleware) AdminOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		payload, ok := auth.GetPayloadFromContext(ctx)
		if !ok {
			response.Unauthorized(c)
			c.Abort()
			return
		}
		sc := auth.NewScope(payload)

		// Check if user has admin role
		if !sc.IsAdmin() {
			m.l.Warnf(ctx, "middleware.AdminOnly: user %s is not admin (role: %s)", sc.UserID, sc.Role)
			response.Forbidden(c)
			c.Abort()
			return
		}

		c.Next()
	}
}
