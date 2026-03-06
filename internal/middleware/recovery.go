package middleware

import (
	"ingest-srv/pkg/discord"
	"ingest-srv/pkg/log"
	"ingest-srv/pkg/response"

	"github.com/gin-gonic/gin"
)

// Recovery recovers from panics and logs the error to Discord.
func Recovery(logger log.Logger, discordClient discord.IDiscord) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				ctx := c.Request.Context()
				logger.Errorf(ctx, "Panic recovered: %v | Method: %s | Path: %s",
					err, c.Request.Method, c.Request.URL.Path)

				response.PanicError(c, err, discordClient)
				c.Abort()
			}
		}()
		c.Next()
	}
}
