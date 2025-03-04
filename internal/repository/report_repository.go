package repository

import (
	"gitlab.ozon.dev/sadsnake2311/homework/internal/domain"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/storage"
)

type ReportRepository interface {
	GetUserOrders(userID string, limit int, status string, offset int) ([]domain.Order, error)
	GetRefundedOrders(limit, offset int) ([]domain.Order, error)
	GetOrderHistory() ([]domain.Order, error)
}

type reportRepository struct {
	reportOrderStorage storage.ReportOrderStorage
}

func NewReportRepository(storage storage.ReportOrderStorage) ReportRepository {
	return &reportRepository{reportOrderStorage: storage}
}

func (r *reportRepository) GetUserOrders(userID string, limit int, status string, offset int) ([]domain.Order, error) {
	return r.reportOrderStorage.GetUserOrders(userID, limit, status, offset)
}

func (r *reportRepository) GetRefundedOrders(limit, offset int) ([]domain.Order, error) {
	return r.reportOrderStorage.GetRefundedOrders(limit, offset)
}

func (r *reportRepository) GetOrderHistory() ([]domain.Order, error) {
	return r.reportOrderStorage.GetOrderHistory()
}
