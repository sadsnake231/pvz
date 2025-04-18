package cache

import (
	"context"
	"log"
	"sync"

	"github.com/redis/go-redis/extra/redisotel/v9"
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

		if err := redisotel.InstrumentTracing(redisClient); err != nil {
			log.Printf("failed to trace redis: %v", err)
		}
	})
	return redisClient
}

type OrderCache interface {
	SetOrder(ctx context.Context, order domain.Order) error
	GetOrder(ctx context.Context, orderID string) (*domain.Order, error)
	GetOrdersBatch(ctx context.Context, orderIDs []string) (map[string]*domain.Order, error)
	DeleteOrder(ctx context.Context, orderID string) error

	GetAllActiveOrderIDs(ctx context.Context) ([]string, error)
	UpdateAllActiveOrders(ctx context.Context, orderIDs []string) error
	GetUserActiveOrders(ctx context.Context, userID string) ([]string, error)
	UpdateUserActiveOrders(ctx context.Context, userID string, orderIDs []string) error
	DeleteUserIndex(ctx context.Context, userID string) error

	GetHistoryOrderIDs(ctx context.Context) ([]string, error)
	AddToHistory(ctx context.Context, orderID string) error
	RemoveFromHistory(ctx context.Context, orderID string) error

	RefreshActiveOrders(ctx context.Context) error
	RefreshHistory(ctx context.Context) error
}
