package service

import (
	"time"

	"gitlab.ozon.dev/sadsnake2311/homework/hw-1/internal/domain"
	"gitlab.ozon.dev/sadsnake2311/homework/hw-1/internal/storage"
)

type StorageService struct {
	orderStorage    	storage.OrderStorage
	userOrderStorage 	storage.UserOrderStorage
	reportOrderStorage    	storage.ReportOrderStorage
}

func NewStorageService(
	orderStorage storage.OrderStorage,
	userOrderStorage storage.UserOrderStorage,
	reportOrderStorage storage.ReportOrderStorage,
) *StorageService {
	return &StorageService{
		orderStorage:     orderStorage,
		userOrderStorage: userOrderStorage,
		reportOrderStorage:    reportOrderStorage,
	}
}

func (s *StorageService) AcceptOrder(order domain.Order) error {
	// Валидация срока
	if order.Expiry.Before(time.Now()) {
		return domain.ErrExpiredOrder
	}

	_, existing, err := s.orderStorage.FindOrderByID(order.ID)
	if err != nil {
		return err
	}
	if existing != nil {
		return domain.ErrDuplicateOrder
	}

	order.Status = domain.StatusStored
	order.UpdatedAt = time.Now()

	return s.orderStorage.SaveOrder(order)
}

func (s *StorageService) ReturnOrder(id string) (string, error) {
	_, order, err := s.orderStorage.FindOrderByID(id)
	if err != nil {
		return "", err
	}

	if order == nil {
		return "", domain.ErrNotFoundOrder
	}

	if !time.Now().After(order.Expiry) {
		return "", domain.ErrNotExpiredOrder
	}

	if order.Status == domain.StatusIssued {
		return "", domain.ErrNotStoredOrder
	}
	return s.orderStorage.DeleteOrder(id)
}

func (s *StorageService) IssueOrders(userID string, orderIDs []string) (string, []string, error){
	return s.userOrderStorage.IssueOrders(userID, orderIDs)
}

func (s *StorageService) RefundOrders(userID string, orderIDs []string) (string, []string, error){
	return s.userOrderStorage.RefundOrders(userID, orderIDs)
}

func (s *StorageService) GetUserOrders(userID string, limit int, status string, offset int) ([]domain.Order, error) {
	return s.reportOrderStorage.GetUserOrders(userID, limit, status, offset)
}

func (s *StorageService) GetRefundedOrders(limit int, offset int) ([]domain.Order, error) {
	return s.reportOrderStorage.GetRefundedOrders(limit, offset)
}

func (s *StorageService) GetOrderHistory() ([]domain.Order, error) {
	return s.reportOrderStorage.GetOrderHistory()
}

