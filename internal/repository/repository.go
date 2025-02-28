package repository

import (
	"time"
	"strings"
	"fmt"

	"gitlab.ozon.dev/sadsnake2311/homework/internal/domain"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/storage"
)

type OrderRepository interface {
	AcceptOrder(order domain.Order) error
	ReturnOrder(id string) (string, error)
	IssueOrders(userID string, orderIDs []string) (string, []string, error)
	RefundOrders(userID string, orderIDs []string) (string, []string, error)
	GetUserOrders(userID string, limit int, status string, offset int) ([]domain.Order, error)
	GetRefundedOrders(limit, offset int) ([]domain.Order, error)
	GetOrderHistory() ([]domain.Order, error)
}

type Repository struct {
	orderStorage     storage.OrderStorage
	userOrderStorage storage.UserOrderStorage
	reportOrderStorage storage.ReportOrderStorage
}

func NewRepository(
	orderStorage storage.OrderStorage,
	userOrderStorage storage.UserOrderStorage,
	reportOrderStorage storage.ReportOrderStorage,
) OrderRepository {
	return &Repository{
		orderStorage:     orderStorage,
		userOrderStorage: userOrderStorage,
		reportOrderStorage: reportOrderStorage,
	}
}

func (r *Repository) AcceptOrder(order domain.Order) error {
	if order.Expiry.Before(time.Now()) {
		return domain.ErrExpiredOrder
	}

	if _, err := r.findOrderByID(order.ID); err == nil {
		return domain.ErrDuplicateOrder
	}

	packaging, err := parsePackaging(string(order.Packaging))
	if err != nil {
		return err
	}

	if !packaging.CheckWeight(order.Weight) {
		return fmt.Errorf("вес заказа слишком велик для выбранной упаковки")
	}

	order.FinalPrice = packaging.CalculatePrice(order.BasePrice)

	order.Status = domain.StatusStored
	order.UpdatedAt = time.Now()

	return r.orderStorage.SaveOrder(order)
}

func (r *Repository) ReturnOrder(id string) (string, error) {
	order, err := r.findOrderByID(id)
	if err != nil {
		return "", err
	}

	if !time.Now().After(order.Expiry) {
		return "", domain.ErrNotExpiredOrder
	}

	if order.Status == domain.StatusIssued {
		return "", domain.ErrNotStoredOrder
	}

	return r.orderStorage.DeleteOrder(id)
}

func (r *Repository) IssueOrders(userID string, orderIDs []string) (string, []string, error) {
	if err := r.validateUser(userID); err != nil {
		return "", nil, err
	}

	for _, orderID := range orderIDs {
		if _, err := r.findOrderByID(orderID); err != nil {
			return "", nil, err
		}
	}

	return r.userOrderStorage.IssueOrders(userID, orderIDs)
}

func (r *Repository) RefundOrders(userID string, orderIDs []string) (string, []string, error) {
	if err := r.validateUser(userID); err != nil {
		return "", nil, err
	}

	for _, orderID := range orderIDs {
		if _, err := r.findOrderByID(orderID); err != nil {
			return "", nil, err
		}
	}

	return r.userOrderStorage.RefundOrders(userID, orderIDs)
}

func (r *Repository) GetUserOrders(userID string, limit int, status string, offset int) ([]domain.Order, error) {
	if err := r.validateUser(userID); err != nil {
		return nil, err
	}

	return r.reportOrderStorage.GetUserOrders(userID, limit, status, offset)
}

func (r *Repository) GetRefundedOrders(limit, offset int) ([]domain.Order, error) {
	return r.reportOrderStorage.GetRefundedOrders(limit, offset)
}

func (r *Repository) GetOrderHistory() ([]domain.Order, error) {
	return r.reportOrderStorage.GetOrderHistory()
}

func (r *Repository) findOrderByID(id string) (*domain.Order, error) {
	_, order, err := r.orderStorage.FindOrderByID(id)
	if err != nil {
		return nil, err
	}
	if order == nil {
		return nil, domain.ErrNotFoundOrder
	}
	return order, nil
}

func (r *Repository) validateUser(userID string) error {
	orders, err := r.reportOrderStorage.GetUserOrders(userID, 1, "", 0)
	if err != nil {
		return err
	}
	if len(orders) == 0 {
		return domain.ErrUserNoOrders
	}
	return nil
}

func parsePackaging(input string) (domain.PackagingStrategy, error) {
	types := strings.Split(input, "+")
	strategies := make([]domain.PackagingStrategy, 0, len(types))

	var mainPackagingCount int
	for _, t := range types {
		switch domain.PackagingType(t) {
			case domain.PackagingType1, domain.PackagingType2:
				mainPackagingCount++
				if mainPackagingCount > 1 {
					return nil, fmt.Errorf("нельзя комбинировать основные типы упаковки")
				}
			case domain.PackagingType3:
			default:
				return nil, fmt.Errorf("неизвестный тип упаковки")
		}
	}

	for _, t := range types {
		switch domain.PackagingType(t) {
			case domain.PackagingType1:
				strategies = append(strategies, domain.Packaging1{})
			case domain.PackagingType2:
				strategies = append(strategies, domain.Packaging2{})
			case domain.PackagingType3:
				strategies = append(strategies, domain.Packaging3{})
		}
	}

	if len(strategies) == 1 {
		return strategies[0], nil
	}

	return domain.CompositePackaging{Strategies: strategies}, nil
}
