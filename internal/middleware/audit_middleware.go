package middleware

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/audit"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/domain"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/metrics"
)

func AuditMiddleware(p *audit.Pipeline) gin.HandlerFunc {
	return func(c *gin.Context) {
		method := c.Request.Method
		path := c.Request.URL.Path

		metrics.HTTPRequestCount.WithLabelValues(method, path).Inc()

		p.SendEvent(domain.EventAPIRequest, map[string]any{
			"method": method,
			"path":   path,
		})

		c.Next()

		status := fmt.Sprintf("%d", c.Writer.Status())

		metrics.HTTPResponseStatusCount.WithLabelValues(status, path).Inc()

		p.SendEvent(domain.EventAPIResponse, map[string]any{
			"status": c.Writer.Status(),
			"path":   c.Request.URL.Path,
		})
	}
}
