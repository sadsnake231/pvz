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
	grpcapi "gitlab.ozon.dev/sadsnake2311/homework/internal/transport/grpc/gen"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type OrderHandler struct {
	grpcapi.UnimplementedOrderHandlerServer
	service  service.OrderService
	pipeline *audit.Pipeline
}

func NewOrderHandler(service service.OrderService, pipeline *audit.Pipeline) *OrderHandler {
	return &OrderHandler{service: service, pipeline: pipeline}
}

func (h *OrderHandler) AcceptOrder(ctx context.Context, req *grpcapi.AcceptOrderRequest) (*grpcapi.AcceptOrderResponse, error) {
	expiry, err := time.Parse("2006-1-02", req.GetExpiry())
	if err != nil {
		metrics.FailedOrderCount.Inc()
		return nil, status.Errorf(codes.InvalidArgument, "неправильный формат времени: %v", err)
	}

	storedAt := time.Now().UTC()
	order := domain.Order{
		ID:          req.GetId(),
		RecipientID: req.GetRecipientId(),
		Expiry:      expiry.Add(24 * time.Hour).UTC(),
		BasePrice:   req.GetBasePrice(),
		Weight:      req.GetWeight(),
		Packaging:   domain.PackagingType(req.GetPackaging()),
		StoredAt:    &storedAt,
	}

	if err := h.service.AcceptOrder(ctx, order); err != nil {
		metrics.FailedOrderCount.Inc()
		return nil, convertOrderError(err)
	}

	h.pipeline.SendEvent(domain.EventStatusChange, map[string]any{
		"order_id": req.GetId(),
		"status":   domain.StatusStored,
	})

	metrics.OrderValueDistribution.Observe(req.GetBasePrice())
	metrics.OrderWeightDistribution.Observe(req.GetWeight())
	metrics.OrdersByStatus.WithLabelValues("stored").Inc()

	return &grpcapi.AcceptOrderResponse{Message: "заказ принят"}, nil
}

func (h *OrderHandler) ReturnOrder(ctx context.Context, req *grpcapi.ReturnOrderRequest) (*grpcapi.ReturnOrderResponse, error) {
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

	metrics.OrderReturns.Inc()
	metrics.OrdersByStatus.WithLabelValues("stored").Dec()
	metrics.OrdersByStatus.WithLabelValues("refunded").Dec()

	return &grpcapi.ReturnOrderResponse{Message: "заказ удален"}, nil
}

func (h *OrderHandler) IssueRefundOrders(ctx context.Context, req *grpcapi.IssueRefundRequest) (*grpcapi.IssueRefundResponse, error) {
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
			metrics.OrdersByStatus.WithLabelValues("stored").Dec()
			metrics.OrdersByStatus.WithLabelValues("issued").Inc()
		case "refund":
			metrics.OrdersByStatus.WithLabelValues("issued").Dec()
			metrics.OrdersByStatus.WithLabelValues("refunded").Inc()
		}
	}

	return &grpcapi.IssueRefundResponse{
		ProcessedOrderIds: result.ProcessedOrderIDs,
		FailedOrderIds:    result.FailedOrderIds,
		Error:             result.Error,
	}, nil
}

func (h *OrderHandler) GetUserOrders(ctx context.Context, req *grpcapi.GetUserOrdersRequest) (*grpcapi.GetUserOrdersResponse, error) {
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

	return &grpcapi.GetUserOrdersResponse{
		Orders:     convertOrdersToPB(domainOrders),
		NextCursor: nextCursor,
	}, nil
}

func (h *OrderHandler) GetRefundedOrders(
	ctx context.Context,
	req *grpcapi.GetRefundedOrdersRequest,
) (*grpcapi.GetRefundedOrdersResponse, error) {
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

	return &grpcapi.GetRefundedOrdersResponse{
		Orders:     convertOrdersToPB(domainOrders),
		NextCursor: nextCursor,
	}, nil
}

func (h *OrderHandler) GetOrderHistory(
	ctx context.Context,
	req *grpcapi.GetOrderHistoryRequest,
) (*grpcapi.GetOrderHistoryResponse, error) {
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

	return &grpcapi.GetOrderHistoryResponse{
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
	req *grpcapi.GetUserActiveOrdersRequest,
) (*grpcapi.GetUserActiveOrdersResponse, error) {
	if req.GetUserId() == "" {
		return nil, status.Error(codes.InvalidArgument, "нужно указать user_id")
	}

	orders, err := h.service.GetUserActiveOrders(ctx, req.GetUserId())
	if err != nil {
		return nil, convertOrderError(err)
	}

	return &grpcapi.GetUserActiveOrdersResponse{
		Orders: convertOrdersToPB(orders),
	}, nil
}

func (h *OrderHandler) GetAllActiveOrders(
	ctx context.Context,
	_ *emptypb.Empty,
) (*grpcapi.GetAllActiveOrdersResponse, error) {
	orders, err := h.service.GetAllActiveOrders(ctx)
	if err != nil {
		return nil, convertOrderError(err)
	}

	return &grpcapi.GetAllActiveOrdersResponse{
		Orders: convertOrdersToPB(orders),
	}, nil
}

func (h *OrderHandler) GetOrderHistoryV2(
	ctx context.Context,
	_ *emptypb.Empty,
) (*grpcapi.GetOrderHistoryV2Response, error) {
	orders, err := h.service.GetOrderHistoryV2(ctx)
	if err != nil {
		return nil, convertOrderError(err)
	}

	return &grpcapi.GetOrderHistoryV2Response{Orders: convertOrdersToPB(orders)}, nil
}

func convertOrdersToPB(orders []domain.Order) []*grpcapi.Order {
	pbOrders := make([]*grpcapi.Order, 0, len(orders))
	for _, o := range orders {
		pbOrder := &grpcapi.Order{
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
