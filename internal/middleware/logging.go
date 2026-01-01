package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hexaend/storage/internal/logger"
)

func LoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()
		method := c.Request.Method
		clientIP := c.ClientIP()

		if raw != "" {
			path = path + "?" + raw
		}

		log := logger.FromContext(c)

		event := log.Info()
		if status >= 500 {
			event = log.Error()
		} else if status >= 400 {
			event = log.Warn()
		}

		event.
			Str("method", method).
			Str("path", path).
			Int("status", status).
			Dur("latency", latency).
			Str("ip", clientIP).
			Int("size", c.Writer.Size()).
			Msg("request completed")
	}
}
