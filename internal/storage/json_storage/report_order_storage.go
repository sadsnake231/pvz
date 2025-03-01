package jsonstorage


import (
	"fmt"
	"sort"

	"gitlab.ozon.dev/sadsnake2311/homework/internal/domain"
)

func (s *JSONOrderStorage) GetUserOrders(userID string, limit int, status string, offset int) ([]domain.Order, error) {
	orders, err := s.readAll()
	if err != nil {
		return nil, fmt.Errorf("ошибка при чтении заказов: %v", err)
	}

	filteredOrders := s.filterOrdersByUser(orders, userID)
	if status != "" {
		filteredOrders = s.filterOrdersByStatus(filteredOrders, status)
	}

	s.sortOrdersByExpiry(filteredOrders)

	if offset > len(filteredOrders) {
		return nil, fmt.Errorf("смещение превышает количество заказов")
	}

	return s.paginateOrders(filteredOrders, limit, offset), nil
}

func (s *JSONOrderStorage) filterOrdersByUser(orders []domain.Order, userID string) []domain.Order {
	filtered := make([]domain.Order, 0)
	for _, order := range orders {
		if order.RecipientID == userID {
			filtered = append(filtered, order)
		}
	}
	return filtered
}

func (s *JSONOrderStorage) filterOrdersByStatus(orders []domain.Order, status string) []domain.Order {
	filtered := make([]domain.Order, 0)
	for _, order := range orders {
		if order.Status == domain.OrderStatus(status) {
			filtered = append(filtered, order)
		}
	}
	return filtered
}

func (s *JSONOrderStorage) sortOrdersByExpiry(orders []domain.Order) {
	sort.Slice(orders, func(i, j int) bool {
		return orders[i].Expiry.After(orders[j].Expiry)
	})
}

func (s *JSONOrderStorage) paginateOrders(orders []domain.Order, limit int, offset int) []domain.Order {
	start := offset
	end := offset + limit
	if end > len(orders) || limit == -1 {
		end = len(orders)
	}
	return orders[start:end]
}

func (s *JSONOrderStorage) GetRefundedOrders(limit int, offset int) ([]domain.Order, error) {


	orders, err := s.readAll()
	if err != nil {
		return nil, fmt.Errorf("ошибка при чтении заказов: %v", err)
	}

	filteredOrders := make([]domain.Order, 0)
	for _, order := range orders {
		if order.Status == domain.StatusRefunded {
			filteredOrders = append(filteredOrders, order)
		}
	}

	sort.Slice(filteredOrders, func(i, j int) bool {
		return filteredOrders[i].UpdatedAt.After(filteredOrders[j].UpdatedAt)
	})

	if offset > len(filteredOrders) {
		return nil, fmt.Errorf("смещение превышает количество заказов")
	}

	start := offset
	end := offset + limit
	if end > len(filteredOrders) || limit == -1 {
		end = len(filteredOrders)
	}

	return filteredOrders[start:end], nil
}

func (s *JSONOrderStorage) GetOrderHistory() ([]domain.Order, error) {


	orders, err := s.readAll()
	if err != nil {
		return nil, fmt.Errorf("ошибка при чтении заказов: %v", err)
	}

	sort.Slice(orders, func(i, j int) bool {
		return orders[i].UpdatedAt.After(orders[j].UpdatedAt)
	})

	return orders, nil
}
