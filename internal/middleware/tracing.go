package middleware

import (
	"ingest-srv/pkg/log"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	XTraceIDHeader = "X-Trace-Id"
)

func Tracing() gin.HandlerFunc {
	return func(c *gin.Context) {
		traceID := c.GetHeader(XTraceIDHeader)
		if traceID == "" {
			traceID = uuid.New().String()
		}

		// Set trace id in context for pkg/log to pick up
		c.Set(log.TraceIDKey, traceID)
		
		// Also set in response header
		c.Header(XTraceIDHeader, traceID)

		c.Next()
	}
}
