package middleware

import (
	"ingest-srv/pkg/locale"

	"github.com/gin-gonic/gin"
)

// Locale returns a middleware that extracts and sets the locale from the request header.
func (m Middleware) Locale() gin.HandlerFunc {
	return func(c *gin.Context) {
		langHeader := c.GetHeader("lang")

		lang := locale.ParseLang(langHeader)

		ctx := c.Request.Context()
		ctx = locale.SetLocaleToContext(ctx, lang)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}
