package handler

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"gitlab.ozon.dev/sadsnake2311/homework/internal/audit"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/domain"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/metrics"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/service"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/transport/grpc/gen/order"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type OrderHandler struct {
	order.UnimplementedOrderHandlerServer
	service  service.OrderService
	pipeline *audit.Pipeline
}

func NewOrderHandler(service service.OrderService, pipeline *audit.Pipeline) *OrderHandler {
	return &OrderHandler{service: service, pipeline: pipeline}
}

func (h *OrderHandler) AcceptOrder(ctx context.Context, req *order.AcceptOrderRequest) (*order.AcceptOrderResponse, error) {
	expiry, err := time.Parse("2006-01-02", req.GetExpiry())
	if err != nil {
		metrics.FailedOrderCount.Inc()
		return nil, status.Errorf(codes.InvalidArgument, "неправильный формат времени: %v", err)
	}

	storedAt := time.Now().UTC()
	orderToAccept := domain.Order{
		ID:          req.GetId(),
		RecipientID: req.GetRecipientId(),
		Expiry:      expiry.Add(24 * time.Hour).UTC(),
		BasePrice:   req.GetBasePrice(),
		Weight:      req.GetWeight(),
		Packaging:   domain.PackagingType(req.GetPackaging()),
		StoredAt:    &storedAt,
	}

	if err := h.service.AcceptOrder(ctx, orderToAccept); err != nil {
		metrics.FailedOrderCount.Inc()
		return nil, convertOrderError(err)
	}

	h.pipeline.SendEvent(domain.EventStatusChange, map[string]any{
		"order_id": req.GetId(),
		"status":   domain.StatusStored,
	})

	metrics.ObserveOrderValue(req.GetBasePrice())
	metrics.ObserveOrderWeight(req.GetWeight())
	metrics.IncOrdersByStatus("stored")

	return &order.AcceptOrderResponse{Message: "заказ принят"}, nil
}

func (h *OrderHandler) ReturnOrder(ctx context.Context, req *order.ReturnOrderRequest) (*order.ReturnOrderResponse, error) {
	if req.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "нужно указать order id")
	}

	if err := h.service.ReturnOrder(ctx, req.GetId()); err != nil {
		return nil, convertOrderError(err)
	}

	h.pipeline.SendEvent(domain.EventStatusChange, map[string]any{
		"order_id": req.GetId(),
		"status":   "Deleted",
	})

	metrics.IncOrderReturns()

	return &order.ReturnOrderResponse{Message: "заказ удален"}, nil
}

func (h *OrderHandler) IssueRefundOrders(ctx context.Context, req *order.IssueRefundRequest) (*order.IssueRefundResponse, error) {
	var (
		result      *service.IssueRefundResponse
		err         error
		orderStatus domain.OrderStatus
	)

	switch req.GetCommand() {
	case "issue":
		result, err = h.service.IssueOrders(ctx, req.GetUserId(), req.GetOrderIds())
		orderStatus = domain.StatusIssued
	case "refund":
		result, err = h.service.RefundOrders(ctx, req.GetUserId(), req.GetOrderIds())
		orderStatus = domain.StatusRefunded
	default:
		return nil, status.Error(codes.InvalidArgument, "неверная команда")
	}

	if err != nil {
		return nil, convertOrderError(err)
	}

	for _, id := range result.ProcessedOrderIDs {
		h.pipeline.SendEvent(domain.EventStatusChange, map[string]any{
			"order_id": id,
			"status":   orderStatus,
		})

		switch orderStatus {
		case "issue":
			metrics.IncOrdersByStatus("issued")
		case "refund":
			metrics.IncOrdersByStatus("refunded")
		}
	}

	return &order.IssueRefundResponse{
		ProcessedOrderIds: result.ProcessedOrderIDs,
		FailedOrderIds:    result.FailedOrderIds,
		Error:             result.Error,
	}, nil
}

func (h *OrderHandler) GetUserOrders(ctx context.Context, req *order.GetUserOrdersRequest) (*order.GetUserOrdersResponse, error) {
	var cursorInt *int
	if cursor := req.GetCursor(); cursor != "" {
		val, err := strconv.Atoi(cursor)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "неверный формат курсора")
		}
		cursorInt = &val
	}

	orders, nextCursor, err := h.service.GetUserOrders(
		ctx,
		req.GetUserId(),
		int(req.GetLimit()),
		cursorInt,
		req.GetStatus(),
	)

	if err != nil {
		return nil, convertOrderError(err)
	}

	domainOrders := convertServiceOrdersToDomain(orders)

	return &order.GetUserOrdersResponse{
		Orders:     convertOrdersToPB(domainOrders),
		NextCursor: nextCursor,
	}, nil
}

func (h *OrderHandler) GetRefundedOrders(
	ctx context.Context,
	req *order.GetRefundedOrdersRequest,
) (*order.GetRefundedOrdersResponse, error) {
	var cursorInt *int
	if cursor := req.GetCursor(); cursor != "" {
		val, err := strconv.Atoi(cursor)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "неверный формат курсора")
		}
		cursorInt = &val
	}

	orders, nextCursor, err := h.service.GetRefundedOrders(
		ctx,
		int(req.GetLimit()),
		cursorInt,
	)

	if err != nil {
		return nil, convertOrderError(err)
	}

	domainOrders := convertServiceOrdersToDomain(orders)

	return &order.GetRefundedOrdersResponse{
		Orders:     convertOrdersToPB(domainOrders),
		NextCursor: nextCursor,
	}, nil
}

func (h *OrderHandler) GetOrderHistory(
	ctx context.Context,
	req *order.GetOrderHistoryRequest,
) (*order.GetOrderHistoryResponse, error) {
	lastUpdatedCursor, err := time.Parse(time.RFC3339, req.GetLastUpdatedCursor())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "неверный формат курсора: %v", err)
	}
	idCursor := int(req.GetIdCursor())

	orders, nextCursor, err := h.service.GetOrderHistory(
		ctx,
		int(req.GetLimit()),
		lastUpdatedCursor,
		idCursor,
	)

	if err != nil {
		return nil, convertOrderError(err)
	}

	domainOrders := convertServiceOrdersToDomain(orders)

	return &order.GetOrderHistoryResponse{
		Orders:     convertOrdersToPB(domainOrders),
		NextCursor: formatCursor(nextCursor),
	}, nil
}

func formatCursor(cursor string) string {
	if cursor == "" {
		return ""
	}
	parts := strings.Split(cursor, ",")
	if len(parts) != 2 {
		return ""
	}
	return fmt.Sprintf("%s,%s", parts[0], parts[1])
}

func (h *OrderHandler) GetUserActiveOrders(
	ctx context.Context,
	req *order.GetUserActiveOrdersRequest,
) (*order.GetUserActiveOrdersResponse, error) {
	if req.GetUserId() == "" {
		return nil, status.Error(codes.InvalidArgument, "нужно указать user_id")
	}

	orders, err := h.service.GetUserActiveOrders(ctx, req.GetUserId())
	if err != nil {
		return nil, convertOrderError(err)
	}

	return &order.GetUserActiveOrdersResponse{
		Orders: convertOrdersToPB(orders),
	}, nil
}

func (h *OrderHandler) GetAllActiveOrders(
	ctx context.Context,
	_ *order.GetAllActiveOrdersRequest,
) (*order.GetAllActiveOrdersResponse, error) {
	orders, err := h.service.GetAllActiveOrders(ctx)
	if err != nil {
		return nil, convertOrderError(err)
	}

	return &order.GetAllActiveOrdersResponse{
		Orders: convertOrdersToPB(orders),
	}, nil
}

func (h *OrderHandler) GetOrderHistoryV2(
	ctx context.Context,
	_ *order.GetOrderHistoryV2Request,
) (*order.GetOrderHistoryV2Response, error) {
	orders, err := h.service.GetOrderHistoryV2(ctx)
	if err != nil {
		return nil, convertOrderError(err)
	}

	return &order.GetOrderHistoryV2Response{Orders: convertOrdersToPB(orders)}, nil
}

func convertOrdersToPB(orders []domain.Order) []*order.Order {
	pbOrders := make([]*order.Order, 0, len(orders))
	for _, o := range orders {
		pbOrder := &order.Order{
			Id:          o.ID,
			RecipientId: o.RecipientID,
			Expiry:      o.Expiry.Format(time.RFC3339),
			BasePrice:   o.BasePrice,
			Weight:      o.Weight,
			Packaging:   string(o.Packaging),
		}

		pbOrders = append(pbOrders, pbOrder)
	}
	return pbOrders
}

func convertOrderError(err error) error {
	switch {
	case errors.Is(err, domain.ErrDuplicateOrder):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, domain.ErrDatabase):
		return status.Error(codes.Internal, err.Error())
	default:
		return status.Error(codes.InvalidArgument, err.Error())
	}
}

func convertServiceOrdersToDomain(orders []service.OrderResponse) []domain.Order {
	domainOrders := make([]domain.Order, 0, len(orders))
	for _, o := range orders {
		expiry, _ := time.Parse(time.RFC3339, o.Expiry)
		storedAt, _ := time.Parse(time.RFC3339, o.StoredAt)

		domainOrders = append(domainOrders, domain.Order{
			ID:          o.ID,
			RecipientID: o.RecipientID,
			Expiry:      expiry,
			BasePrice:   o.BasePrice,
			Weight:      o.Weight,
			Packaging:   o.Packaging,
			StoredAt:    &storedAt,
		})
	}
	return domainOrders
}
