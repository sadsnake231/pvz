package middleware

import (
	"net/http"
	"time"

	"gitlab.ozon.dev/sadsnake2311/homework/internal/metrics"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
)

func MetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		duration := time.Since(start).Seconds()
		status := http.StatusText(c.Writer.Status())

		labels := prometheus.Labels{
			"endpoint": c.FullPath(),
			"method":   c.Request.Method,
			"status":   status,
		}

		metrics.APIResponseTime.With(labels).Observe(duration)
	}
}
