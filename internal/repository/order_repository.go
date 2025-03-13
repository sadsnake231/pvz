package repository

import (
	"context"
	"errors"
	"time"

	"gitlab.ozon.dev/sadsnake2311/homework/internal/domain"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/repository/utils_repository"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/storage"
	"go.uber.org/zap"
)

type OrderRepository interface {
	AcceptOrder(ctx context.Context, order domain.Order) error
	ReturnOrder(ctx context.Context, id string) error
	FindOrderByID(ctx context.Context, id string) (*domain.Order, error)
}

type orderRepository struct {
	orderStorage storage.OrderStorage
	logger       *zap.Logger
}

func NewOrderRepository(storage storage.OrderStorage, logger *zap.Logger) OrderRepository {
	return &orderRepository{
		orderStorage: storage,
		logger:       logger,
	}
}

func (r *orderRepository) AcceptOrder(ctx context.Context, order domain.Order) error {
	if order.Expiry.Before(time.Now()) {
		return domain.ErrExpiredOrder
	}

	existing, err := r.orderStorage.FindOrderByID(ctx, order.ID)
	if err != nil && !errors.Is(err, domain.ErrNotFoundOrder) {
		r.logger.Error("Не удалось проверить существование заказа", zap.Error(err))
		return domain.ErrDatabase
	}
	if existing != nil {
		return domain.ErrDuplicateOrder
	}

	packaging, err := utils_repository.ParsePackaging(string(order.Packaging))
	if err != nil {
		return err
	}

	if !packaging.CheckWeight(order.Weight) {
		return domain.ErrInvalidWeight
	}

	order.PackagePrice = packaging.CalculatePrice()
	now := time.Now().UTC()
	order.StoredAt = &now

	err = r.orderStorage.SaveOrder(ctx, order)
	if err != nil {
		r.logger.Error("Не удалось сохранить заказ", zap.Error(err))
		return domain.ErrDatabase
	}
	return nil
}

func (r *orderRepository) ReturnOrder(ctx context.Context, id string) error {
	order, err := r.orderStorage.FindOrderByID(ctx, id)
	if err != nil && !errors.Is(err, domain.ErrNotFoundOrder) {
		r.logger.Error("Не удалось проверить существование заказа", zap.Error(err))
		return domain.ErrDatabase
	}

	if order.Status() != domain.StatusStored {
		return domain.ErrNotStoredOrder
	}

	if order.Expiry.After(time.Now()) {
		return domain.ErrNotExpiredOrder
	}

	if err := r.orderStorage.DeleteOrder(ctx, id); err != nil {
		r.logger.Error("Не удалось удалить заказ", zap.String("orderID", id), zap.Error(err))
		return domain.ErrDatabase
	}

	return nil
}

func (r *orderRepository) FindOrderByID(ctx context.Context, id string) (*domain.Order, error) {
	order, err := r.orderStorage.FindOrderByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return order, nil
}
