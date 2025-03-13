package storage

import (
	"context"
	"time"

	"gitlab.ozon.dev/sadsnake2311/homework/internal/domain"
)

type OrderStorage interface {
	SaveOrder(ctx context.Context, order domain.Order) error
	FindOrderByID(ctx context.Context, id string) (*domain.Order, error)
	DeleteOrder(ctx context.Context, id string) error
}

type UserOrderStorage interface {
	IssueOrders(ctx context.Context, userID string, orderIDs []string) (domain.ProcessedOrders, error)
	RefundOrders(ctx context.Context, userID string, orderIDs []string) (domain.ProcessedOrders, error)
}

type ReportOrderStorage interface {
	GetUserOrders(ctx context.Context, userID string, limit int, cursor *int, status string) ([]domain.Order, string, error)
	GetRefundedOrders(ctx context.Context, limit int, offset *int) ([]domain.Order, string, error)
	GetOrderHistory(ctx context.Context, limit int, lastUpdatedCursor time.Time, idCursor int) ([]domain.Order, string, error)
}

type AuthStorage interface {
	CreateUser(ctx context.Context, user *domain.User) error
	GetUserByEmail(ctx context.Context, email string) (*domain.User, error)
}
