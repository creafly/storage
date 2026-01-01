package middleware

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hexaend/storage/internal/infra/monitoring"
)

func PrometheusMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}

		c.Next()

		duration := time.Since(start).Seconds()
		status := strconv.Itoa(c.Writer.Status())

		monitoring.HTTPRequestsTotal.WithLabelValues(c.Request.Method, path, status).Inc()
		monitoring.HTTPRequestDuration.WithLabelValues(c.Request.Method, path).Observe(duration)
	}
}
