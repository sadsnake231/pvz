package cache

import (
	"context"
	"sync"

	"github.com/redis/go-redis/v9"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/domain"
)

var (
	redisClient *redis.Client
	once        sync.Once
)

func GetRedisClient(addr, password string) *redis.Client {
	once.Do(func() {
		redisClient = redis.NewClient(&redis.Options{
			Addr:     addr,
			Password: password,
		})
	})
	return redisClient
}

type OrderCache interface {
	SetOrder(ctx context.Context, order domain.Order) error
	GetOrder(ctx context.Context, orderID string) (*domain.Order, error)
	DeleteOrder(ctx context.Context, orderID string) error

	GetAllActiveOrderIDs(ctx context.Context) ([]string, error)
	UpdateAllActiveIndex(ctx context.Context, orderIDs []string) error
	GetUserActiveOrders(ctx context.Context, userID string) ([]string, error)
	UpdateUserIndex(ctx context.Context, userID string, orderIDs []string) error
	DeleteUserIndex(ctx context.Context, userID string) error

	GetHistoryOrderIDs(ctx context.Context) ([]string, error)
	UpdateHistoryIndex(ctx context.Context, orderIDs []string) error
	DeleteFromHistory(ctx context.Context, orderID string) error
}
