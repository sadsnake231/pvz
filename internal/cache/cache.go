package cache

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/domain"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/repository/reportrepo"
)

const (
	userActiveKeyPrefix = "active_orders:"
	allActiveKey        = "all_active_orders"
	historyKey          = "order_history"
	orderKeyPrefix      = "order:"
)

type RedisCache struct {
	client     *redis.Client
	reportRepo reportrepo.ReportRepository
}

func NewRedisCache(client *redis.Client, reportRepo reportrepo.ReportRepository) *RedisCache {
	return &RedisCache{client: client, reportRepo: reportRepo}
}

func (c *RedisCache) SetOrder(ctx context.Context, order domain.Order) error {
	key := orderKeyPrefix + order.ID
	data, err := json.Marshal(order)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, key, data, calculateOrderTTL(order)).Err()
}

func (c *RedisCache) GetOrder(ctx context.Context, orderID string) (*domain.Order, error) {
	data, err := c.client.Get(ctx, orderKeyPrefix+orderID).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var order domain.Order
	if err := json.Unmarshal(data, &order); err != nil {
		return nil, err
	}
	return &order, nil
}

func (c *RedisCache) GetOrdersBatch(ctx context.Context, orderIDs []string) (map[string]*domain.Order, error) {
	pipe := c.client.Pipeline()
	cmds := make(map[string]*redis.StringCmd, len(orderIDs))
	for _, id := range orderIDs {
		cmds[id] = pipe.Get(ctx, orderKeyPrefix+id)
	}
	if _, err := pipe.Exec(ctx); err != nil && err != redis.Nil {
		return nil, err
	}
	result := make(map[string]*domain.Order)
	for id, cmd := range cmds {
		data, err := cmd.Bytes()
		if err != nil {
			continue
		}
		var order domain.Order
		if err := json.Unmarshal(data, &order); err != nil {
			continue
		}
		result[id] = &order
	}
	return result, nil
}

func (c *RedisCache) DeleteOrder(ctx context.Context, orderID string) error {
	return c.client.Del(ctx, orderKeyPrefix+orderID).Err()
}

func (c *RedisCache) GetUserActiveOrders(ctx context.Context, userID string) ([]string, error) {
	return c.client.SMembers(ctx, userActiveKeyPrefix+userID).Result()
}

func (c *RedisCache) UpdateUserActiveOrders(ctx context.Context, userID string, orderIDs []string) error {
	pipe := c.client.Pipeline()
	key := userActiveKeyPrefix + userID
	pipe.Del(ctx, key)
	if len(orderIDs) > 0 {
		pipe.SAdd(ctx, key, orderIDs)
		pipe.Expire(ctx, key, 14*24*time.Hour)
	}
	_, err := pipe.Exec(ctx)
	return err
}

func (c *RedisCache) DeleteUserIndex(ctx context.Context, userID string) error {
	return c.client.Del(ctx, userActiveKeyPrefix+userID).Err()
}

func (c *RedisCache) GetAllActiveOrderIDs(ctx context.Context) ([]string, error) {
	return c.client.SMembers(ctx, allActiveKey).Result()
}

func (c *RedisCache) UpdateAllActiveOrders(ctx context.Context, orderIDs []string) error {
	pipe := c.client.Pipeline()
	pipe.Del(ctx, allActiveKey)
	if len(orderIDs) > 0 {
		pipe.SAdd(ctx, allActiveKey, orderIDs)
		pipe.Expire(ctx, allActiveKey, 14*24*time.Hour)
	}
	_, err := pipe.Exec(ctx)
	return err
}

func (c *RedisCache) GetHistoryOrderIDs(ctx context.Context) ([]string, error) {
	return c.client.SMembers(ctx, historyKey).Result()
}

func (c *RedisCache) AddToHistory(ctx context.Context, orderID string) error {
	return c.client.SAdd(ctx, historyKey, orderID).Err()
}

func (c *RedisCache) RemoveFromHistory(ctx context.Context, orderID string) error {
	return c.client.SRem(ctx, historyKey, orderID).Err()
}

func (c *RedisCache) RefreshActiveOrders(ctx context.Context) error {
	orderIDs, err := c.reportRepo.GetAllActiveOrderIDs(ctx)
	if err != nil {
		return err
	}
	return c.UpdateAllActiveOrders(ctx, orderIDs)
}

func (c *RedisCache) RefreshHistory(ctx context.Context) error {
	orderIDs, err := c.reportRepo.GetHistoryOrderIDs(ctx)
	if err != nil {
		return err
	}
	pipe := c.client.Pipeline()
	pipe.Del(ctx, historyKey)
	if len(orderIDs) > 0 {
		pipe.SAdd(ctx, historyKey, orderIDs)
	}
	_, err = pipe.Exec(ctx)
	return err
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
