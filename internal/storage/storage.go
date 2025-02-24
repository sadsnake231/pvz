package storage

import "gitlab.ozon.dev/sadsnake2311/homework/internal/domain"

type OrderStorage interface {
	SaveOrder	(order domain.Order) error
	FindOrderByID	(id string) (int, *domain.Order, error)
	DeleteOrder	(id string) (string, error)
}

type UserOrderStorage interface {
	IssueOrders 	(userID string, orderID []string) (string, []string, error)
	RefundOrders	(userID string, orderID []string) (string, []string, error)
}

type ReportOrderStorage interface {
	GetUserOrders 		(userID string, limit int, status string, offset int) ([]domain.Order, error)
	GetRefundedOrders 	(limit, offset int) ([]domain.Order, error)
	GetOrderHistory		()([]domain.Order, error)
}
