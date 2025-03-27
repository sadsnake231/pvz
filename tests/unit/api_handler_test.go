package api

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/api"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/domain"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/service"
)

type MockOrderService struct {
	mock.Mock
}

func (m *MockOrderService) AcceptOrder(ctx context.Context, order domain.Order) error {
	args := m.Called(ctx, order)
	return args.Error(0)
}

func (m *MockOrderService) ReturnOrder(ctx context.Context, orderID string) error {
	args := m.Called(ctx, orderID)
	return args.Error(0)
}

func (m *MockOrderService) IssueOrders(ctx context.Context, userID string, orderIDs []string) (*service.IssueRefundResponse, error) {
	args := m.Called(ctx, userID, orderIDs)
	return args.Get(0).(*service.IssueRefundResponse), args.Error(1)
}

func (m *MockOrderService) RefundOrders(ctx context.Context, userID string, orderIDs []string) (*service.IssueRefundResponse, error) {
	args := m.Called(ctx, userID, orderIDs)
	return args.Get(0).(*service.IssueRefundResponse), args.Error(1)
}

func (m *MockOrderService) GetUserOrders(ctx context.Context, userID string, limit int, cursor *int, status string) ([]service.OrderResponse, string, error) {
	args := m.Called(ctx, userID, limit, cursor, status)
	return args.Get(0).([]service.OrderResponse), args.String(1), args.Error(2)
}

func (m *MockOrderService) GetRefundedOrders(ctx context.Context, limit int, cursor *int) ([]service.OrderResponse, string, error) {
	args := m.Called(ctx, limit, cursor)
	return args.Get(0).([]service.OrderResponse), args.String(1), args.Error(2)
}

func (m *MockOrderService) GetOrderHistory(ctx context.Context, limit int, lastUpdatedCursor time.Time, idCursor int) ([]service.OrderResponse, string, error) {
	args := m.Called(ctx, limit, lastUpdatedCursor, idCursor)
	return args.Get(0).([]service.OrderResponse), args.String(1), args.Error(2)
}

func TestAPIHandler_AcceptOrder_Success(t *testing.T) {
	mockService := new(MockOrderService)
	handler := api.NewAPIHandler(mockService)

	//expiryTime := time.Now().Add(24 * time.Hour).UTC()
	mockService.On("AcceptOrder", mock.Anything, mock.MatchedBy(func(order domain.Order) bool {
		return order.ID == "123" && order.RecipientID == "user1"
	})).Return(nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(
		"POST",
		"/orders",
		bytes.NewBufferString(`{
			"id": "123",
			"recipient_id": "user1",
			"expiry": "2025-04-01",
			"base_price": 100,
			"weight": 2.5,
			"packaging": "коробка"
		}`),
	)
	c.Request.Header.Set("Content-Type", "application/json")

	handler.AcceptOrder(c)

	assert.Equal(t, http.StatusCreated, w.Code)
	mockService.AssertExpectations(t)
}

func TestAPIHandler_AcceptOrder_InvalidExpiry(t *testing.T) {
	handler := api.NewAPIHandler(nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(
		"POST",
		"/orders",
		bytes.NewBufferString(`{
			"id": "123",
			"recipient_id": "user1",
			"expiry": "invalid-date",
			"base_price": 100,
			"weight": 2.5,
			"packaging": "коробка"
		}`),
	)
	c.Request.Header.Set("Content-Type", "application/json")

	handler.AcceptOrder(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "неверный формат времени")
}

func TestAPIHandler_ReturnOrder_ServiceError(t *testing.T) {
	mockService := new(MockOrderService)
	handler := api.NewAPIHandler(mockService)

	mockService.On("ReturnOrder", mock.Anything, "123").Return(domain.ErrDatabase)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("DELETE", "/orders/123/return", nil)
	c.AddParam("id", "123")

	handler.ReturnOrder(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockService.AssertExpectations(t)
}

func TestAPIHandler_GetUserOrders_Success(t *testing.T) {
	mockService := new(MockOrderService)
	handler := api.NewAPIHandler(mockService)

	mockService.On("GetUserOrders", mock.Anything, "user1", 10, (*int)(nil), "").
		Return([]service.OrderResponse{
			{ID: "123", RecipientID: "user1"},
		}, "next-cursor", nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/reports/user1/orders?limit=10", nil)
	c.Params = gin.Params{{Key: "user_id", Value: "user1"}}

	handler.GetUserOrders(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"orders":`)
	assert.Contains(t, w.Body.String(), `"next_cursor":"next-cursor"`)
	mockService.AssertExpectations(t)
}

func TestAPIHandler_IssueRefundOrders_InvalidCommand(t *testing.T) {
	handler := api.NewAPIHandler(nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(
		"PUT",
		"/actions/issues_refunds",
		bytes.NewBufferString(`{
			"command": "invalid",
			"user_id": "user1",
			"order_ids": ["123"]
		}`),
	)
	c.Request.Header.Set("Content-Type", "application/json")

	handler.IssueRefundOrders(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "неверная команда")
}

func TestAPIHandler_GetRefundedOrders_InvalidLimit(t *testing.T) {
	handler := api.NewAPIHandler(nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/reports/refunded?limit=invalid", nil)

	handler.GetRefundedOrders(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "неверный формат лимита")
}
