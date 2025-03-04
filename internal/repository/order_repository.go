package repository

import (
	"fmt"
	"time"

	"gitlab.ozon.dev/sadsnake2311/homework/internal/domain"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/repository/utils_repository"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/storage"
)

type OrderRepository interface {
	AcceptOrder(order domain.Order) error
	ReturnOrder(id string) (string, error)
	FindOrderByID(id string) (*domain.Order, error)
}

type orderRepository struct {
	orderStorage storage.OrderStorage
}

func NewOrderRepository(storage storage.OrderStorage) OrderRepository {
	return &orderRepository{orderStorage: storage}
}

func (r *orderRepository) AcceptOrder(order domain.Order) error {
	if order.Expiry.Before(time.Now()) {
		return domain.ErrExpiredOrder
	}

	if _, err := r.FindOrderByID(order.ID); err == nil {
		return domain.ErrDuplicateOrder
	}

	packaging, err := utils_repository.ParsePackaging(string(order.Packaging))
	if err != nil {
		return err
	}

	if !packaging.CheckWeight(order.Weight) {
		return fmt.Errorf("вес заказа слишком велик для выбранной упаковки")
	}

	order.PackagePrice = packaging.CalculatePrice()

	order.Status = domain.StatusStored
	order.UpdatedAt = time.Now()

	return r.orderStorage.SaveOrder(order)
}

func (r *orderRepository) ReturnOrder(id string) (string, error) {
	order, err := r.FindOrderByID(id)
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

func (r *orderRepository) FindOrderByID(id string) (*domain.Order, error) {
	_, order, err := r.orderStorage.FindOrderByID(id)
	if err != nil {
		return nil, err
	}
	if order == nil {
		return nil, domain.ErrNotFoundOrder
	}
	return order, nil
}
