package userorderrepo

import (
	"context"
	"errors"

	"gitlab.ozon.dev/sadsnake2311/homework/internal/domain"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/storage"
	"go.uber.org/zap"
)

type UserOrderRepository interface {
	IssueOrders(ctx context.Context, userID string, orderIDs []string) (domain.ProcessedOrders, error)
	RefundOrders(ctx context.Context, userID string, orderIDs []string) (domain.ProcessedOrders, error)
}

type userOrderRepository struct {
	userOrderStorage storage.UserOrderStorage
	logger           *zap.SugaredLogger
}

func NewUserOrderRepository(storage storage.UserOrderStorage, logger *zap.SugaredLogger) UserOrderRepository {
	return &userOrderRepository{userOrderStorage: storage, logger: logger}
}

func (r *userOrderRepository) IssueOrders(ctx context.Context, userID string, orderIDs []string) (domain.ProcessedOrders, error) {
	result, err := r.userOrderStorage.IssueOrders(ctx, userID, orderIDs)
	if err != nil {
		if errors.Is(err, domain.ErrDatabase) {
			r.logger.Error("Не удалось выдать заказ",
				zap.Error(err),
				zap.String("userID", userID),
				zap.Strings("orderIDs", orderIDs),
			)
			return domain.ProcessedOrders{}, domain.ErrDatabase
		}
		return domain.ProcessedOrders{}, err
	}
	return result, nil
}

func (r *userOrderRepository) RefundOrders(ctx context.Context, userID string, orderIDs []string) (domain.ProcessedOrders, error) {
	result, err := r.userOrderStorage.RefundOrders(ctx, userID, orderIDs)
	if err != nil {
		if errors.Is(err, domain.ErrDatabase) {
			r.logger.Error("Не удалось вернуть заказ",
				zap.Error(err),
				zap.String("userID", userID),
				zap.Strings("orderIDs", orderIDs),
			)
			return domain.ProcessedOrders{}, domain.ErrDatabase
		}
		return domain.ProcessedOrders{}, err
	}
	return result, nil
}
