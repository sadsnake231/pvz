package repository

import (
	"gitlab.ozon.dev/sadsnake2311/homework/internal/domain"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/storage"
)

type UserOrderRepository interface {
	IssueOrders(userID string, orderIDs []string) domain.ProcessedOrders
	RefundOrders(userID string, orderIDs []string) domain.ProcessedOrders
}

type userOrderRepository struct {
	userOrderStorage storage.UserOrderStorage
}

func NewUserOrderRepository(storage storage.UserOrderStorage) UserOrderRepository {
	return &userOrderRepository{userOrderStorage: storage}
}

func (r *userOrderRepository) IssueOrders(userID string, orderIDs []string) domain.ProcessedOrders {
	return r.userOrderStorage.IssueOrders(userID, orderIDs)
}

func (r *userOrderRepository) RefundOrders(userID string, orderIDs []string) domain.ProcessedOrders {
	return r.userOrderStorage.RefundOrders(userID, orderIDs)
}
