package service

import (
	"fmt"
	"strings"

	"gitlab.ozon.dev/sadsnake2311/homework/internal/domain"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/repository"
)

type StorageService interface {
	AcceptOrder(order domain.Order) (string, error)
	AcceptOrdersFromJSON(orders []domain.Order) (string, error)
	ReturnOrder(orderID string) (string, error)
	IssueOrders(userID string, orderIDs []string) (string, error)
	RefundOrders(userID string, orderIDs []string) (string, error)
	GetUserOrders(userID string, limit int, showStored bool) ([]domain.Order, error)
	GetRefundedOrders(limit, offset int) ([]domain.Order, error)
	GetOrderHistory() ([]domain.Order, error)
}

type storageService struct {
	orderRepo     repository.OrderRepository
	userOrderRepo repository.UserOrderRepository
	reportRepo    repository.ReportRepository
}

func NewStorageService(
	orderRepo repository.OrderRepository,
	userOrderRepo repository.UserOrderRepository,
	reportRepo repository.ReportRepository,
) StorageService {
	return &storageService{
		orderRepo:     orderRepo,
		userOrderRepo: userOrderRepo,
		reportRepo:    reportRepo,
	}
}

func (s *storageService) AcceptOrder(order domain.Order) (string, error) {
	if err := s.orderRepo.AcceptOrder(order); err != nil {
		return "", err
	}
	return "Заказ принят!", nil
}

func (s *storageService) AcceptOrdersFromJSON(orders []domain.Order) (string, error) {
	for _, order := range orders {
		if _, err := s.AcceptOrder(order); err != nil {
			return "", fmt.Errorf("ошибка при обработке заказа %s: %v", order.ID, err)
		}
	}
	return "Заказы через JSON приняты!", nil
}

func (s *storageService) ReturnOrder(orderID string) (string, error) {
	id, err := s.orderRepo.ReturnOrder(orderID)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("Заказ %s возвращен курьеру!", id), nil
}

func (s *storageService) IssueOrders(userID string, orderIDs []string) (string, error) {
	for _, orderID := range orderIDs {
		if _, err := s.orderRepo.FindOrderByID(orderID); err != nil {
			return "", err
		}
	}

	processedOrders := s.userOrderRepo.IssueOrders(userID, orderIDs)

	return fmt.Sprintf("Успешно обработанные заказы: %s", strings.Join(processedOrders.OrderIDs, ", ")), processedOrders.Error
}

func (s *storageService) RefundOrders(userID string, orderIDs []string) (string, error) {
	for _, orderID := range orderIDs {
		if _, err := s.orderRepo.FindOrderByID(orderID); err != nil {
			return "", err
		}
	}

	processedOrders := s.userOrderRepo.RefundOrders(userID, orderIDs)

	return fmt.Sprintf("Успешно обработанные заказы: %s", strings.Join(processedOrders.OrderIDs, ", ")), processedOrders.Error
}

func (s *storageService) GetUserOrders(userID string, limit int, showStored bool) ([]domain.Order, error) {
	status := ""
	if showStored {
		status = string(domain.StatusStored)
	}
	return s.reportRepo.GetUserOrders(userID, limit, status, 0)
}

func (s *storageService) GetRefundedOrders(limit, offset int) ([]domain.Order, error) {
	return s.reportRepo.GetRefundedOrders(limit, offset)
}

func (s *storageService) GetOrderHistory() ([]domain.Order, error) {
	return s.reportRepo.GetOrderHistory()
}
