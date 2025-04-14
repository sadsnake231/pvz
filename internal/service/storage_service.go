package service

import (
	"context"
	"time"

	"gitlab.ozon.dev/sadsnake2311/homework/internal/cache"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/domain"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/metrics"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/repository/orderrepo"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/repository/reportrepo"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/repository/userorderrepo"
	"go.uber.org/zap"
)

type OrderService interface {
	AcceptOrder(ctx context.Context, order domain.Order) error
	ReturnOrder(ctx context.Context, orderID string) error
	IssueOrders(ctx context.Context, userID string, orderIDs []string) (*IssueRefundResponse, error)
	RefundOrders(ctx context.Context, userID string, orderIDs []string) (*IssueRefundResponse, error)
	GetUserOrders(ctx context.Context, userID string, limit int, cursor *int, status string) ([]OrderResponse, string, error)
	GetRefundedOrders(ctx context.Context, limit int, cursor *int) ([]OrderResponse, string, error)
	GetOrderHistory(ctx context.Context, limit int, lastUpdatedCursor time.Time, idCursor int) ([]OrderResponse, string, error)
	GetUserActiveOrders(ctx context.Context, userID string) ([]domain.Order, error)
	GetAllActiveOrders(ctx context.Context) ([]domain.Order, error)
	GetOrderHistoryV2(ctx context.Context) ([]domain.Order, error)

	CacheRefresh(ctx context.Context)
	InitCache(ctx context.Context)
}

type orderService struct {
	orderRepo     orderrepo.OrderRepository
	userOrderRepo userorderrepo.UserOrderRepository
	reportRepo    reportrepo.ReportRepository
	cache         cache.OrderCache
	logger        *zap.SugaredLogger
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
	cache *cache.RedisCache,
	logger *zap.SugaredLogger,
) OrderService {
	return &orderService{
		orderRepo:     orderRepo,
		userOrderRepo: userOrderRepo,
		reportRepo:    reportRepo,
		cache:         cache,
		logger:        logger,
	}
}

func (s *orderService) AcceptOrder(ctx context.Context, order domain.Order) error {
	if err := s.orderRepo.AcceptOrder(ctx, order); err != nil {
		return err
	}
	go func() {
		cacheCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		s.cache.SetOrder(cacheCtx, order)
		s.cache.AddToHistory(cacheCtx, order.ID)
		activeIDs, _ := s.cache.GetUserActiveOrders(cacheCtx, order.RecipientID)
		activeIDs = append(activeIDs, order.ID)
		s.cache.UpdateUserActiveOrders(cacheCtx, order.RecipientID, activeIDs)
	}()
	return nil
}

func (s *orderService) ReturnOrder(ctx context.Context, orderID string) error {
	order, err := s.orderRepo.FindOrderByID(ctx, orderID)
	if err != nil {
		return err
	}
	if err := s.orderRepo.ReturnOrder(ctx, orderID); err != nil {
		return err
	}
	go func() {
		cacheCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		s.cache.DeleteOrder(cacheCtx, orderID)
		activeIDs, _ := s.cache.GetUserActiveOrders(cacheCtx, order.RecipientID)
		newActiveIDs := make([]string, 0, len(activeIDs))
		for _, id := range activeIDs {
			if id != orderID {
				newActiveIDs = append(newActiveIDs, id)
			}
		}
		if len(newActiveIDs) > 0 {
			s.cache.UpdateUserActiveOrders(cacheCtx, order.RecipientID, newActiveIDs)
		} else {
			s.cache.DeleteUserIndex(cacheCtx, order.RecipientID)
		}
	}()
	return nil
}

func (s *orderService) IssueOrders(ctx context.Context, userID string, orderIDs []string) (*IssueRefundResponse, error) {
	result, err := s.userOrderRepo.IssueOrders(ctx, userID, orderIDs)
	if err != nil {
		return &IssueRefundResponse{}, err
	}

	for _, orderID := range result.OrderIDs {
		order, err := s.cache.GetOrder(ctx, orderID)
		if err != nil {
			s.logger.Errorf("failed to get order %s from cache: %v", orderID, err)
			continue
		}
		if order == nil {
			continue
		}

		order.IssuedAt = new(time.Time)
		*order.IssuedAt = time.Now()

		if err := s.cache.SetOrder(ctx, *order); err != nil {
			s.logger.Errorf("failed to update order %s in cache: %v", order.ID, err)
		}
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

	for _, orderID := range result.OrderIDs {
		if err := s.cache.DeleteOrder(ctx, orderID); err != nil {
			s.logger.Errorf("failed to delete order %s from cache: %v", orderID, err)
		}
		if err := s.cache.RemoveFromHistory(ctx, orderID); err != nil {
			s.logger.Errorf("failed to delete order %s from cache: %v", orderID, err)
		}
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

func (s *orderService) GetOrderHistoryV2(ctx context.Context) ([]domain.Order, error) {
	orderIDs, err := s.cache.GetHistoryOrderIDs(ctx)
	if err != nil {
		orderIDs = []string{}
	}
	orders, _ := s.cache.GetOrdersBatch(ctx, orderIDs)
	result := make([]domain.Order, 0, len(orders))
	for _, order := range orders {
		if order != nil {
			result = append(result, *order)
		}
	}
	if len(result) == 0 {
		result, _, err = s.reportRepo.GetOrderHistory(ctx, 0, time.Time{}, 0)
		return result, err
	}
	return result, nil
}

func (s *orderService) GetUserActiveOrders(ctx context.Context, userID string) ([]domain.Order, error) {
	startTime := time.Now()
	defer func() {
		metrics.DBQueryDuration.WithLabelValues("GetUserActiveOrders").Observe(time.Since(startTime).Seconds())
	}()

	orderIDs, err := s.cache.GetUserActiveOrders(ctx, userID)
	if err != nil {
		metrics.CacheOperations.WithLabelValues("GetHistoryIDs", "error").Inc()
		s.logger.Errorf("failed to get the orders of user %s from cache: %v", userID, err)
	} else {
		metrics.CacheOperations.WithLabelValues("GetHistoryIDs", "success").Inc()
	}

	if len(orderIDs) == 0 {
		metrics.CacheMisses.WithLabelValues("history").Inc()
		orderIDs, err = s.reportRepo.GetUserActiveOrderIDs(ctx, userID)
		if err != nil {
			return nil, err
		}

		if err := s.cache.UpdateUserActiveOrders(ctx, userID, orderIDs); err != nil {
			s.logger.Errorf("failed to update in cache: %v", err)
		}
	} else {
		metrics.CacheHits.WithLabelValues("history").Inc()
	}

	var orders []domain.Order
	for _, orderID := range orderIDs {
		order, err := s.cache.GetOrder(ctx, orderID)
		if err != nil {
			s.logger.Errorf("failed to get order %s from cache: %v", orderID, err)
			continue
		}
		if order == nil {
			order, err = s.orderRepo.FindOrderByID(ctx, orderID)
			if err != nil {
				continue
			}
			if err := s.cache.SetOrder(ctx, *order); err != nil {
				s.logger.Errorf("failed to cache order %s: %v", order.ID, err)
			}
		}
		orders = append(orders, *order)
	}

	return orders, nil
}

func (s *orderService) GetAllActiveOrders(ctx context.Context) ([]domain.Order, error) {
	startTime := time.Now()
	defer func() {
		metrics.DBQueryDuration.WithLabelValues("GetAllActiveOrders").Observe(time.Since(startTime).Seconds())
	}()

	orderIDs, err := s.cache.GetAllActiveOrderIDs(ctx)
	if err != nil {
		metrics.CacheOperations.WithLabelValues("GetHistoryIDs", "error").Inc()
		s.logger.Errorf("failed to get the orders from cache: %v", err)
	} else {
		metrics.CacheOperations.WithLabelValues("GetHistoryIDs", "success").Inc()
	}

	if len(orderIDs) == 0 {
		metrics.CacheMisses.WithLabelValues("history").Inc()
		orderIDs, err = s.reportRepo.GetAllActiveOrderIDs(ctx)
		if err != nil {
			return nil, err
		}

		if err := s.cache.UpdateAllActiveOrders(ctx, orderIDs); err != nil {
			s.logger.Errorf("failed to update in cache: %v", err)
		}
	} else {
		metrics.CacheHits.WithLabelValues("history").Inc()
	}

	var orders []domain.Order
	for _, orderID := range orderIDs {
		order, err := s.cache.GetOrder(ctx, orderID)
		if err != nil {
			s.logger.Errorf("failed to get order %s from cache: %v", orderID, err)
			continue
		}
		if order == nil {
			order, err = s.orderRepo.FindOrderByID(ctx, orderID)
			if err != nil {
				continue
			}
			if err := s.cache.SetOrder(ctx, *order); err != nil {
				s.logger.Errorf("failed to cache order %s: %v", order.ID, err)
			}
		}
		orders = append(orders, *order)
	}

	return orders, nil
}

func (s *orderService) InitCache(ctx context.Context) {
	startTime := time.Now()

	if activeIDs, err := s.reportRepo.GetAllActiveOrderIDs(ctx); err == nil {
		if err := s.cache.UpdateAllActiveOrders(ctx, activeIDs); err != nil {
			s.logger.Errorf("failed to init the cache: %v", err)
		}

		if orders, err := s.orderRepo.FindOrdersByIDs(ctx, activeIDs); err == nil {
			for _, order := range orders {
				if err := s.cache.SetOrder(ctx, *order); err != nil {
					s.logger.Errorf("failed to cache order %s: %v", order.ID, err)
				}
			}
		}
	}

	if historyIDs, err := s.reportRepo.GetHistoryOrderIDs(ctx); err == nil {
		if err := s.cache.RefreshHistory(ctx); err != nil {
			s.logger.Errorf("failed to init the cache: %v", err)
		}

		if orders, err := s.orderRepo.FindOrdersByIDs(ctx, historyIDs); err == nil {
			for _, order := range orders {
				if err := s.cache.SetOrder(ctx, *order); err != nil {
					s.logger.Errorf("failed to cache order %s: %v", order.ID, err)
				}
			}
		}
	}

	s.logger.Infof("Кэш инициализирован, заняло %v", time.Since(startTime))
}

func (s *orderService) CacheRefresh(ctx context.Context) {
	activeTicker := time.NewTicker(5 * time.Minute)
	historyTicker := time.NewTicker(30 * time.Minute)
	defer func() {
		activeTicker.Stop()
		historyTicker.Stop()
	}()
	for {
		select {
		case <-ctx.Done():
			return
		case <-activeTicker.C:
			s.cache.RefreshActiveOrders(ctx)
		case <-historyTicker.C:
			s.cache.RefreshHistory(ctx)
		}
	}
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
