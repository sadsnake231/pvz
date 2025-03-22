package middleware

import (
	"github.com/gin-gonic/gin"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/audit"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/domain"
)

func AuditMiddleware(p *audit.Pipeline) gin.HandlerFunc {
	return func(c *gin.Context) {
		p.DbPool.ApiChan <- domain.NewEvent(domain.EventAPIRequest, map[string]any{
			"method": c.Request.Method,
			"path":   c.Request.URL.Path,
		})
		p.StdoutPool.ApiChan <- domain.NewEvent(domain.EventAPIRequest, map[string]any{
			"method": c.Request.Method,
			"path":   c.Request.URL.Path,
		})

		c.Next()

		p.DbPool.ApiChan <- domain.NewEvent(domain.EventAPIResponse, map[string]any{
			"status": c.Writer.Status(),
			"path":   c.Request.URL.Path,
		})
		p.StdoutPool.ApiChan <- domain.NewEvent(domain.EventAPIResponse, map[string]any{
			"status": c.Writer.Status(),
			"path":   c.Request.URL.Path,
		})
	}
}
