package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/smap-hcmut/shared-libs/go/response"
)

// InternalAuth validates the internal key from X-Internal-Key header
func (m Middleware) InternalAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.GetHeader("X-Internal-Key")
		if key == "" || key != m.InternalKey {
			response.Unauthorized(c)
			c.Abort()
			return
		}
		c.Next()
	}
}
