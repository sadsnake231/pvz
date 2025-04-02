package api

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/audit"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/domain"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/service"
)

type APIHandler struct {
	service  service.OrderService
	pipeline *audit.Pipeline
}

func NewAPIHandler(service service.OrderService, pipeline *audit.Pipeline) *APIHandler {
	return &APIHandler{service: service, pipeline: pipeline}
}

type AcceptOrderRequest struct {
	ID          string               `json:"id" binding:"required"`
	RecipientID string               `json:"recipient_id" binding:"required"`
	Expiry      string               `json:"expiry" binding:"required"`
	BasePrice   float64              `json:"base_price" binding:"required"`
	Weight      float64              `json:"weight" binding:"required"`
	Packaging   domain.PackagingType `json:"packaging" binding:"required"`
}

type IssueRefundRequest struct {
	Command  string   `json:"command" binding:"required"`
	UserID   string   `json:"user_id" binding:"required"`
	OrderIDs []string `json:"order_ids" binding:"required"`
}

func (h *APIHandler) AcceptOrder(c *gin.Context) {
	var req AcceptOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("%v: %v", domain.ErrWrongJSON, err)})
		return
	}

	expiry, err := time.Parse("2006-01-02", req.Expiry)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "неверный формат времени"})
		return
	}

	storedAt := time.Now().UTC()
	order := domain.Order{
		ID:          req.ID,
		RecipientID: req.RecipientID,
		Expiry:      expiry.Add(24 * time.Hour).UTC(),
		BasePrice:   req.BasePrice,
		Weight:      req.Weight,
		Packaging:   req.Packaging,
		StoredAt:    &storedAt,
	}

	err = h.service.AcceptOrder(c.Request.Context(), order)
	if err != nil {
		if errors.Is(err, domain.ErrDatabase) {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if errors.Is(err, domain.ErrDuplicateOrder) {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	h.pipeline.SendEvent(domain.EventStatusChange, map[string]any{
		"order_id": req.ID,
		"status":   domain.StatusStored,
	})
	c.JSON(http.StatusCreated, gin.H{"message": "заказ принят"})
}

func (h *APIHandler) ReturnOrder(c *gin.Context) {
	orderID := c.Param("id")
	if orderID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "нужно указать order id"})
		return
	}

	err := h.service.ReturnOrder(c.Request.Context(), orderID)
	if err != nil {
		if errors.Is(err, domain.ErrDatabase) {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	h.pipeline.SendEvent(domain.EventStatusChange, map[string]any{
		"order_id": orderID,
		"status":   "Deleted",
	})
	c.JSON(http.StatusOK, gin.H{"message": "заказ удален"})
}

func (h *APIHandler) IssueRefundOrders(c *gin.Context) {
	var req IssueRefundRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("%v: %v", domain.ErrWrongJSON, err)})
		return
	}

	var result *service.IssueRefundResponse
	var err error

	var status domain.OrderStatus

	switch req.Command {
	case "issue":
		result, err = h.service.IssueOrders(c.Request.Context(), req.UserID, req.OrderIDs)
		status = domain.StatusIssued
	case "refund":
		result, err = h.service.RefundOrders(c.Request.Context(), req.UserID, req.OrderIDs)
		status = domain.StatusRefunded
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "неверная команда"})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	for _, id := range result.ProcessedOrderIDs {
		h.pipeline.SendEvent(domain.EventStatusChange, map[string]any{
			"order_id": id,
			"status":   status,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"processed_order_ids": result.ProcessedOrderIDs,
		"failed_order_ids":    result.FailedOrderIds,
		"error":               result.Error,
	})

}

func (h *APIHandler) GetUserOrders(c *gin.Context) {
	userID := c.Param("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "требуется указать user id"})
		return
	}

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

	status := c.Query("status")

	orders, nextCursor, err := h.service.GetUserOrders(c.Request.Context(), userID, limit, cursorInt, status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	response := gin.H{"orders": orders}
	if nextCursor != "" {
		response["next_cursor"] = nextCursor
	}
	c.JSON(http.StatusOK, response)
}

func (h *APIHandler) GetRefundedOrders(c *gin.Context) {
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

	orders, nextCursor, err := h.service.GetRefundedOrders(c.Request.Context(), limit, cursorInt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	response := gin.H{"orders": orders}
	if nextCursor != "" {
		response["next_cursor"] = nextCursor
	}
	c.JSON(http.StatusOK, response)
}

func (h *APIHandler) GetOrderHistory(c *gin.Context) {
	limit := 10
	if limitParam := c.Query("limit"); limitParam != "" {
		var err error
		limit, err = strconv.Atoi(limitParam)
		if err != nil || limit < 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "неверный лимит"})
			return
		}
	}

	cursor := c.Query("cursor")
	var (
		lastUpdatedCursor time.Time
		idCursor          int
	)
	if cursor != "" {
		parts := strings.Split(cursor, ",")
		if len(parts) != 2 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "неверный формат курсора"})
			return
		}

		var err error
		lastUpdatedCursor, err = time.Parse(time.RFC3339Nano, parts[0])
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "неверное время курсора"})
			return
		}

		idCursor, err = strconv.Atoi(parts[1])
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "неверное id курсора"})
			return
		}
	}

	orders, nextCursor, err := h.service.GetOrderHistory(c.Request.Context(), limit, lastUpdatedCursor, idCursor)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	response := gin.H{"orders": orders}
	if nextCursor != "" {
		response["next_cursor"] = nextCursor
	}
	c.JSON(http.StatusOK, response)
}

func (h *APIHandler) GetUserActiveOrders(c *gin.Context) {
	userID := c.Param("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "требуется указать user id"})
		return
	}

	orders, err := h.service.GetUserActiveOrders(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"orders": orders})
}

func (h *APIHandler) GetAllActiveOrders(c *gin.Context) {

	orders, err := h.service.GetAllActiveOrders(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"orders": orders})
}

func (h *APIHandler) GetOrderHistoryV2(c *gin.Context) {

	orders, err := h.service.GetOrderHistoryV2(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"orders": orders})
}

func (h *APIHandler) GetMetrics(c *gin.Context) {
	handler := promhttp.Handler()
	handler.ServeHTTP(c.Writer, c.Request)
}
