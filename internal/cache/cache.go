package cache

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/domain"
)

const (
	userActiveKeyPrefix = "active_orders:"
	allActiveKey        = "all_active_orders"
	historyKey          = "order_history"
	orderKeyPrefix      = "order:"
)

type RedisCache struct {
	client *redis.Client
}

func NewRedisCache(client *redis.Client) *RedisCache {
	return &RedisCache{client: client}
}

func (c *RedisCache) SetOrder(ctx context.Context, order domain.Order) error {
	key := orderKeyPrefix + order.ID
	data, err := json.Marshal(order)
	if err != nil {
		return err
	}

	ttl := calculateOrderTTL(order)
	return c.client.Set(ctx, key, data, ttl).Err()
}

func (c *RedisCache) GetOrder(ctx context.Context, orderID string) (*domain.Order, error) {
	key := orderKeyPrefix + orderID
	data, err := c.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, domain.ErrCache
	}

	var order domain.Order
	if err := json.Unmarshal(data, &order); err != nil {
		return nil, err
	}
	return &order, nil
}

func (c *RedisCache) DeleteOrder(ctx context.Context, orderID string) error {
	return c.client.Del(ctx, orderKeyPrefix+orderID).Err()
}

func (c *RedisCache) GetUserActiveOrders(ctx context.Context, userID string) ([]string, error) {
	key := userActiveKeyPrefix + userID
	return c.getIndex(ctx, key)
}

func (c *RedisCache) UpdateUserIndex(ctx context.Context, userID string, orderIDs []string) error {
	key := userActiveKeyPrefix + userID
	return c.setIndex(ctx, key, orderIDs, 24*time.Hour)
}

func (c *RedisCache) DeleteUserIndex(ctx context.Context, userID string) error {
	return c.client.Del(ctx, userActiveKeyPrefix+userID).Err()
}

func (c *RedisCache) GetAllActiveOrderIDs(ctx context.Context) ([]string, error) {
	return c.getIndex(ctx, allActiveKey)
}

func (c *RedisCache) UpdateAllActiveIndex(ctx context.Context, orderIDs []string) error {
	return c.setIndex(ctx, allActiveKey, orderIDs, 24*time.Hour)
}

func (c *RedisCache) GetHistoryOrderIDs(ctx context.Context) ([]string, error) {
	return c.getIndex(ctx, historyKey)
}

func (c *RedisCache) UpdateHistoryIndex(ctx context.Context, orderIDs []string) error {
	return c.setIndex(ctx, historyKey, orderIDs, 0)
}

func (c *RedisCache) DeleteFromHistory(ctx context.Context, orderID string) error {
	ids, err := c.getIndex(ctx, historyKey)
	if err != nil {
		return err
	}

	updatedIDs := removeFromSlice(ids, orderID)
	return c.setIndex(ctx, historyKey, updatedIDs, 0)
}

func removeFromSlice(slice []string, target string) []string {
	var result []string
	for _, v := range slice {
		if v != target {
			result = append(result, v)
		}
	}
	return result
}

func (c *RedisCache) Ping(ctx context.Context) error {
	return c.client.Ping(ctx).Err()
}

func (c *RedisCache) getOrdersByIndex(ctx context.Context, indexKey string) ([]domain.Order, error) {
	ids, err := c.getIndex(ctx, indexKey)
	if err != nil {
		return nil, err
	}

	var orders []domain.Order
	for _, id := range ids {
		order, err := c.GetOrder(ctx, id)
		if err != nil {
			continue
		}
		if order != nil {
			orders = append(orders, *order)
		}
	}
	return orders, nil
}

func (c *RedisCache) getIndex(ctx context.Context, key string) ([]string, error) {
	data, err := c.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return []string{}, nil
	}
	if err != nil {
		return nil, err
	}

	var ids []string
	if err := json.Unmarshal(data, &ids); err != nil {
		return nil, err
	}
	return ids, nil
}

func (c *RedisCache) setIndex(ctx context.Context, key string, ids []string, ttl time.Duration) error {
	data, err := json.Marshal(ids)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, key, data, ttl).Err()
}

func calculateOrderTTL(order domain.Order) time.Duration {
	switch {
	case order.RefundedAt != nil:
		return 0
	case order.IssuedAt != nil:
		return time.Until(order.IssuedAt.Add(48 * time.Hour))
	case order.StoredAt != nil:
		return time.Until(order.Expiry)
	default:
		return 24 * time.Hour
	}
}
