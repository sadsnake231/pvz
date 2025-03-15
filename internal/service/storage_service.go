package service

import (
	"context"
	"time"

	"gitlab.ozon.dev/sadsnake2311/homework/internal/domain"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/repository/orderrepo"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/repository/reportrepo"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/repository/userorderrepo"
)

type OrderService interface {
	AcceptOrder(ctx context.Context, order domain.Order) error
	ReturnOrder(ctx context.Context, orderID string) error
	IssueOrders(ctx context.Context, userID string, orderIDs []string) (*IssueRefundResponse, error)
	RefundOrders(ctx context.Context, userID string, orderIDs []string) (*IssueRefundResponse, error)
	GetUserOrders(ctx context.Context, userID string, limit int, cursor *int, status string) ([]OrderResponse, string, error)
	GetRefundedOrders(ctx context.Context, limit int, cursor *int) ([]OrderResponse, string, error)
	GetOrderHistory(ctx context.Context, limit int, lastUpdatedCursor time.Time, idCursor int) ([]OrderResponse, string, error)
}

type orderService struct {
	orderRepo     orderrepo.OrderRepository
	userOrderRepo userorderrepo.UserOrderRepository
	reportRepo    reportrepo.ReportRepository
}

type OrderResponse struct {
	ID          string               `json:"id"`
	RecipientID string               `json:"recipient_id"`
	Expiry      string               `json:"expiry"`
	BasePrice   float64              `json:"base_price"`
	Weight      float64              `json:"weight"`
	Packaging   domain.PackagingType `json:"packaging"`
	Status      string               `json:"status"`
	StoredAt    string               `json:"stored_at,omitempty"`
	IssuedAt    string               `json:"issued_at,omitempty"`
	RefundedAt  string               `json:"refunded_at,omitempty"`
}

type IssueRefundResponse struct {
	ProcessedOrderIDs []string `json:"processed_order_ids"`
	FailedOrderIds    []string `json:"failed_order_ids"`
	Error             string   `json:"error,omitempty"`
}

func NewOrderService(
	orderRepo orderrepo.OrderRepository,
	userOrderRepo userorderrepo.UserOrderRepository,
	reportRepo reportrepo.ReportRepository,
) OrderService {
	return &orderService{
		orderRepo:     orderRepo,
		userOrderRepo: userOrderRepo,
		reportRepo:    reportRepo,
	}
}

func (s *orderService) AcceptOrder(ctx context.Context, order domain.Order) error {
	return s.orderRepo.AcceptOrder(ctx, order)
}

func (s *orderService) ReturnOrder(ctx context.Context, orderID string) error {
	return s.orderRepo.ReturnOrder(ctx, orderID)
}

func (s *orderService) IssueOrders(ctx context.Context, userID string, orderIDs []string) (*IssueRefundResponse, error) {
	result, err := s.userOrderRepo.IssueOrders(ctx, userID, orderIDs)
	if err != nil {
		return &IssueRefundResponse{}, err
	}

	return &IssueRefundResponse{
		ProcessedOrderIDs: result.OrderIDs,
		FailedOrderIds:    result.Failed,
		Error:             errorToString(result.Error),
	}, nil
}

func (s *orderService) RefundOrders(ctx context.Context, userID string, orderIDs []string) (*IssueRefundResponse, error) {
	result, err := s.userOrderRepo.RefundOrders(ctx, userID, orderIDs)
	if err != nil {
		return &IssueRefundResponse{}, err
	}

	return &IssueRefundResponse{
		ProcessedOrderIDs: result.OrderIDs,
		FailedOrderIds:    result.Failed,
		Error:             errorToString(result.Error),
	}, nil
}

func (s *orderService) GetUserOrders(
	ctx context.Context,
	userID string,
	limit int,
	cursor *int,
	status string,
) ([]OrderResponse, string, error) {
	orders, nextCursor, err := s.reportRepo.GetUserOrders(ctx, userID, limit, cursor, status)
	if err != nil {
		return nil, "", err
	}

	return s.mapOrdersToResponses(orders), nextCursor, nil
}

func (s *orderService) GetRefundedOrders(
	ctx context.Context,
	limit int,
	cursor *int,
) ([]OrderResponse, string, error) {
	orders, nextCursor, err := s.reportRepo.GetRefundedOrders(ctx, limit, cursor)
	if err != nil {
		return nil, "", err
	}
	return s.mapOrdersToResponses(orders), nextCursor, nil
}

func (s *orderService) GetOrderHistory(
	ctx context.Context,
	limit int,
	lastUpdatedCursor time.Time,
	idCursor int,
) ([]OrderResponse, string, error) {
	orders, nextCursor, err := s.reportRepo.GetOrderHistory(ctx, limit, lastUpdatedCursor, idCursor)
	if err != nil {
		return nil, "", err
	}

	return s.mapOrdersToResponses(orders), nextCursor, nil
}

func (s *orderService) mapOrderToResponse(order *domain.Order) *OrderResponse {
	resp := &OrderResponse{
		ID:          order.ID,
		RecipientID: order.RecipientID,
		Expiry:      order.Expiry.Format(time.RFC3339),
		BasePrice:   order.BasePrice,
		Weight:      order.Weight,
		Packaging:   order.Packaging,
		Status:      string(order.Status()),
	}

	if order.StoredAt != nil {
		resp.StoredAt = order.StoredAt.Format(time.RFC3339)
	}
	if order.IssuedAt != nil {
		resp.IssuedAt = order.IssuedAt.Format(time.RFC3339)
	}
	if order.RefundedAt != nil {
		resp.RefundedAt = order.RefundedAt.Format(time.RFC3339)
	}

	return resp
}

func (s *orderService) mapOrdersToResponses(orders []domain.Order) []OrderResponse {
	responses := make([]OrderResponse, 0, len(orders))
	for _, order := range orders {
		responses = append(responses, *s.mapOrderToResponse(&order))
	}
	return responses
}

func errorToString(err error) string {
	if err != nil {
		return err.Error()
	}
	return ""
}
