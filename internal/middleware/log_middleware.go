package middleware

import (
	"bytes"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func LogRequestBody(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == http.MethodPost || c.Request.Method == http.MethodPut || c.Request.Method == http.MethodDelete {
			var buf bytes.Buffer
			tee := io.TeeReader(c.Request.Body, &buf)

			body, err := io.ReadAll(tee)
			if err != nil {
				logger.Error("Логгирование запроса не удалось", zap.Error(err))
			}

			c.Request.Body = io.NopCloser(&buf)

			logger.Info("Request body",
				zap.String("method", c.Request.Method),
				zap.String("path", c.Request.URL.Path),
				zap.String("body", string(body)),
			)
		}

		c.Next()
	}
}
