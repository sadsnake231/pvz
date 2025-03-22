package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/service"
)

type AuditHandler struct {
	service service.AuditService
}

func NewAuditHandler(service service.AuditService) *AuditHandler {
	return &AuditHandler{service: service}
}

func (h *AuditHandler) GetLogs(c *gin.Context) {
	limit := 10
	if limitParam := c.Query("limit"); limitParam != "" {
		var err error
		limit, err = strconv.Atoi(limitParam)
		if err != nil || limit < 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "неверный формат лимита"})
			return
		}
	}

	cursor := c.Query("cursor")
	var cursorInt *int
	if cursor != "" {
		cursorVal, err := strconv.Atoi(cursor)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "неверный формат курсора"})
			return
		}
		cursorInt = &cursorVal
	}

	logs, nextCursor, err := h.service.GetLogs(c.Request.Context(), limit, cursorInt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"logs": logs, "cursor": nextCursor})
}
