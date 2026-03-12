package middleware

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/smap-hcmut/shared-libs/go/discord"
	"github.com/smap-hcmut/shared-libs/go/log"
	"github.com/smap-hcmut/shared-libs/go/response"
)

// Recovery recovers from panics and logs the error to Discord.
func Recovery(logger log.Logger, discordClient discord.IDiscord) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				ctx := c.Request.Context()
				logger.Errorf(ctx, "Panic recovered: %v | Method: %s | Path: %s",
					err, c.Request.Method, c.Request.URL.Path)

				response.Error(c, fmt.Errorf("%v", err), discordClient)
				c.Abort()
			}
		}()
		c.Next()
	}
}
