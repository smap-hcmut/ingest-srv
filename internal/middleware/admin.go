package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/smap-hcmut/shared-libs/go/response"
	"github.com/smap-hcmut/shared-libs/go/scope"
)

// AdminOnly is a middleware that checks if the user has admin role
// This middleware should be used after Auth() middleware
func (m Middleware) AdminOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		payload, ok := scope.GetPayloadFromContext(ctx)
		if !ok {
			response.Unauthorized(c)
			c.Abort()
			return
		}
		sc := scope.NewScope(payload)

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

// OptionalAdmin is a middleware that sets isAdmin flag but doesn't block non-admin users
// Useful for endpoints that have different behavior for admin vs regular users
func (m Middleware) OptionalAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		payload, ok := scope.GetPayloadFromContext(ctx)
		if !ok {
			response.Unauthorized(c)
			c.Abort()
			return
		}
		sc := scope.NewScope(payload)

		// Set isAdmin flag in context
		c.Set("isAdmin", sc.IsAdmin())
		c.Next()
	}
}
