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

	if err := s.cache.SetOrder(ctx, order); err != nil {
		s.logger.Errorf("Не смог закэшить заказ %s: %v", order.ID, err)
	}

	userOrderIDs, err := s.cache.GetUserActiveOrders(ctx, order.RecipientID)
	if err != nil {
		s.logger.Errorf("Не смог получить заказы в кэше: %v", err)
		userOrderIDs = []string{}
	}
	userOrderIDs = append(userOrderIDs, order.ID)

	if err := s.cache.UpdateUserIndex(ctx, order.RecipientID, userOrderIDs); err != nil {
		s.logger.Errorf("Не смог обновить заказы в кэше: %v", err)
	}

	allActiveOrders, err := s.cache.GetAllActiveOrderIDs(ctx)
	if err != nil {
		s.logger.Errorf("Не смог получить заказы в кэше: %v", err)
		allActiveOrders = []string{}
	}
	allActiveOrders = append(allActiveOrders, order.ID)

	if err := s.cache.UpdateAllActiveIndex(ctx, allActiveOrders); err != nil {
		s.logger.Errorf("Не смог обновить заказы в кэше: %v", err)
	}

	historyOrders, err := s.cache.GetHistoryOrderIDs(ctx)
	if err != nil {
		s.logger.Errorf("Не смог получить заказы из кэша: %v", err)
		historyOrders = []string{}
	}
	historyOrders = append(historyOrders, order.ID)

	if err := s.cache.UpdateHistoryIndex(ctx, historyOrders); err != nil {
		s.logger.Errorf("Не смог обновить заказы в кэше: %v", err)
	}

	return nil
}

func (s *orderService) ReturnOrder(ctx context.Context, orderID string) error {
	if err := s.orderRepo.ReturnOrder(ctx, orderID); err != nil {
		return err
	}

	if err := s.cache.DeleteOrder(ctx, orderID); err != nil {
		s.logger.Errorf("Не смог удалить из кэша: %v", err)
	}

	if err := s.cache.DeleteFromHistory(ctx, orderID); err != nil {
		s.logger.Errorf("Не смог удалить из кэша: %v", err)
	}

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
			s.logger.Errorf("Не смог получить заказ %s из кэша: %v", orderID, err)
			continue
		}
		if order == nil {
			continue
		}

		order.IssuedAt = new(time.Time)
		*order.IssuedAt = time.Now()

		if err := s.cache.SetOrder(ctx, *order); err != nil {
			s.logger.Errorf("Не смог обновить заказ %s в кэше: %v", order.ID, err)
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
			s.logger.Errorf("Не смог удалить заказ %s из кэша: %v", orderID, err)
		}
		if err := s.cache.DeleteFromHistory(ctx, orderID); err != nil {
			s.logger.Errorf("Не смог удалить заказ %s из кэша: %v", orderID, err)
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
	startTime := time.Now()
	defer func() {
		metrics.DBQueryDuration.WithLabelValues("GetOrderHistory").Observe(time.Since(startTime).Seconds())
	}()

	orderIDs, err := s.cache.GetHistoryOrderIDs(ctx)
	if err != nil {
		metrics.CacheOperations.WithLabelValues("GetHistoryIDs", "error").Inc()
		s.logger.Errorf("Не смог получить заказы из кэша: %v", err)
	} else {
		metrics.CacheOperations.WithLabelValues("GetHistoryIDs", "success").Inc()
	}

	if len(orderIDs) == 0 {
		metrics.CacheMisses.WithLabelValues("history").Inc()

		orderIDs, err = s.reportRepo.GetHistoryOrderIDs(ctx)
		if err != nil {
			return nil, err
		}

		if err := s.cache.UpdateHistoryIndex(ctx, orderIDs); err != nil {
			s.logger.Errorf("Не смог обновить в кэше: %v", err)
		}
	} else {
		metrics.CacheHits.WithLabelValues("history").Inc()
	}

	var orders []domain.Order
	for _, orderID := range orderIDs {
		order, err := s.cache.GetOrder(ctx, orderID)
		if err != nil {
			s.logger.Errorf("Не смог получить заказ %s из кэша: %v", orderID, err)
			continue
		}
		if order == nil {
			order, err = s.orderRepo.FindOrderByID(ctx, orderID)
			if err != nil {
				continue
			}
			if err := s.cache.SetOrder(ctx, *order); err != nil {
				s.logger.Errorf("Не смог закэшить заказ %s: %v", order.ID, err)
			}
		}
		orders = append(orders, *order)
	}

	return orders, nil
}

func (s *orderService) GetUserActiveOrders(ctx context.Context, userID string) ([]domain.Order, error) {
	startTime := time.Now()
	defer func() {
		metrics.DBQueryDuration.WithLabelValues("GetUserActiveOrders").Observe(time.Since(startTime).Seconds())
	}()

	orderIDs, err := s.cache.GetUserActiveOrders(ctx, userID)
	if err != nil {
		metrics.CacheOperations.WithLabelValues("GetHistoryIDs", "error").Inc()
		s.logger.Errorf("Не смог получить заказы юзера %s из кэша: %v", userID, err)
	} else {
		metrics.CacheOperations.WithLabelValues("GetHistoryIDs", "success").Inc()
	}

	if len(orderIDs) == 0 {
		metrics.CacheMisses.WithLabelValues("history").Inc()
		orderIDs, err = s.reportRepo.GetUserActiveOrderIDs(ctx, userID)
		if err != nil {
			return nil, err
		}

		if err := s.cache.UpdateUserIndex(ctx, userID, orderIDs); err != nil {
			s.logger.Errorf("Не смог обновить в кэше: %v", err)
		}
	} else {
		metrics.CacheHits.WithLabelValues("history").Inc()
	}

	var orders []domain.Order
	for _, orderID := range orderIDs {
		order, err := s.cache.GetOrder(ctx, orderID)
		if err != nil {
			s.logger.Errorf("Не смог получить заказ %s из кэша: %v", orderID, err)
			continue
		}
		if order == nil {
			order, err = s.orderRepo.FindOrderByID(ctx, orderID)
			if err != nil {
				continue
			}
			if err := s.cache.SetOrder(ctx, *order); err != nil {
				s.logger.Errorf("Не смог закэшировать заказ %s: %v", order.ID, err)
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
		s.logger.Errorf("Не смог получить заказы из кэша: %v", err)
	} else {
		metrics.CacheOperations.WithLabelValues("GetHistoryIDs", "success").Inc()
	}

	if len(orderIDs) == 0 {
		metrics.CacheMisses.WithLabelValues("history").Inc()
		orderIDs, err = s.reportRepo.GetAllActiveOrderIDs(ctx)
		if err != nil {
			return nil, err
		}

		if err := s.cache.UpdateAllActiveIndex(ctx, orderIDs); err != nil {
			s.logger.Errorf("Не смог обновить в кэше: %v", err)
		}
	} else {
		metrics.CacheHits.WithLabelValues("history").Inc()
	}

	var orders []domain.Order
	for _, orderID := range orderIDs {
		order, err := s.cache.GetOrder(ctx, orderID)
		if err != nil {
			s.logger.Errorf("Не смог получить заказ %s из кэша: %v", orderID, err)
			continue
		}
		if order == nil {
			order, err = s.orderRepo.FindOrderByID(ctx, orderID)
			if err != nil {
				continue
			}
			if err := s.cache.SetOrder(ctx, *order); err != nil {
				s.logger.Errorf("Не смог закэшировать заказ %s: %v", order.ID, err)
			}
		}
		orders = append(orders, *order)
	}

	return orders, nil
}

func (s *orderService) InitCache(ctx context.Context) {
	startTime := time.Now()
	historyIDs, err := s.reportRepo.GetHistoryOrderIDs(ctx)
	if err != nil {
		s.logger.Errorf("Не смог загрузить историю заказов из БД: %v", err)
	} else {
		if err := s.cache.UpdateHistoryIndex(ctx, historyIDs); err != nil {
			s.logger.Errorf("Не смог обновить в кэше: %v", err)
		}
	}

	activeIDs, err := s.reportRepo.GetAllActiveOrderIDs(ctx)
	if err != nil {
		s.logger.Errorf("Не смог загрузить активные заказы из БД: %v", err)
	} else {
		if err := s.cache.UpdateAllActiveIndex(ctx, activeIDs); err != nil {
			s.logger.Errorf("Не смог обновить в кэше: %v", err)
		}
	}

	orders, err := s.reportRepo.GetAllOrders(ctx)
	if err != nil {
		s.logger.Errorf("Не смог загрузить заказы из БД: %v", err)
		return
	}

	for _, order := range orders {
		if err := s.cache.SetOrder(ctx, order); err != nil {
			s.logger.Errorf("Не смог закэшировать заказ %s: %v", order.ID, err)
		}
	}

	s.logger.Infof("Кэш инициализирован, заняло %v", time.Since(startTime))
}

func (s *orderService) CacheRefresh(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("Остановка рефрешера кэша")
			return
		case <-ticker.C:
			s.refreshHistoryCache(ctx)
		}
	}
}

func (s *orderService) refreshHistoryCache(ctx context.Context) {
	startTime := time.Now()
	s.logger.Info("Рефреш кэша...")

	ids, err := s.reportRepo.GetHistoryOrderIDs(ctx)
	if err != nil {
		return
	}

	if err := s.cache.UpdateHistoryIndex(ctx, ids); err != nil {
		return
	}

	s.logger.Infof("Рефреш кэша окончен, заняло %v", time.Since(startTime))
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
