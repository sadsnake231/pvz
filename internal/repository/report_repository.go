package repository

import (
	"context"
	"time"

	"gitlab.ozon.dev/sadsnake2311/homework/internal/domain"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/storage"
	"go.uber.org/zap"
)

type ReportRepository interface {
	GetUserOrders(ctx context.Context, userID string, limit int, cursor *int, status string) ([]domain.Order, string, error)
	GetRefundedOrders(ctx context.Context, limit int, cursor *int) ([]domain.Order, string, error)
	GetOrderHistory(ctx context.Context, limit int, lastUpdatedCursor time.Time, idCursor int) ([]domain.Order, string, error)
}

type reportRepository struct {
	reportOrderStorage storage.ReportOrderStorage
	logger             *zap.Logger
}

func NewReportRepository(storage storage.ReportOrderStorage, logger *zap.Logger) ReportRepository {
	return &reportRepository{reportOrderStorage: storage, logger: logger}
}

func (r *reportRepository) GetUserOrders(
	ctx context.Context,
	userID string,
	limit int,
	cursor *int,
	status string,
) ([]domain.Order, string, error) {
	res, newCursor, err := r.reportOrderStorage.GetUserOrders(ctx, userID, limit, cursor, status)
	if err != nil {
		r.logger.Error("ошибка вывода заказов", zap.Error(err))
		return res, newCursor, domain.ErrDatabase
	}
	return res, newCursor, err
}

func (r *reportRepository) GetRefundedOrders(
	ctx context.Context,
	limit int,
	cursor *int,
) ([]domain.Order, string, error) {
	res, newCursor, err := r.reportOrderStorage.GetRefundedOrders(ctx, limit, cursor)
	if err != nil {
		r.logger.Error("ошибка вывода заказов", zap.Error(err))
		return res, newCursor, domain.ErrDatabase
	}

	return res, newCursor, err
}

func (r *reportRepository) GetOrderHistory(
	ctx context.Context,
	limit int,
	lastUpdatedCursor time.Time,
	idCursor int,
) ([]domain.Order, string, error) {
	res, newCursor, err := r.reportOrderStorage.GetOrderHistory(ctx, limit, lastUpdatedCursor, idCursor)
	if err != nil {
		r.logger.Error("ошибка вывода заказов", zap.Error(err))
		return res, newCursor, domain.ErrDatabase
	}

	return res, newCursor, err
}
