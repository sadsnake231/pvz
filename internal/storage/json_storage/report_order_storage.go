package jsonstorage


import (
	"fmt"
	"sort"

	"gitlab.ozon.dev/sadsnake2311/homework/hw-1/internal/domain"
)

func (s *JSONOrderStorage) GetUserOrders(userID string, limit int, status string, offset int) ([]domain.Order, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	orders, err := s.readAll()
	if err != nil {
		return nil, fmt.Errorf("ошибка при чтении заказов: %v", err)
	}

	filteredOrders := make([]domain.Order, 0)
	for _, order := range orders {
		if order.RecipientID == userID {
			filteredOrders = append(filteredOrders, order)
		}
	}

	if status != "" {
		temp := make([]domain.Order, 0)
		for _, order := range filteredOrders {
			if order.Status == domain.OrderStatus(status) {
				temp = append(temp, order)
			}
		}
		filteredOrders = temp
	}

	sort.Slice(filteredOrders, func(i, j int) bool {
		return filteredOrders[i].Expiry.After(filteredOrders[j].Expiry)
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

func (s *JSONOrderStorage) GetRefundedOrders(limit int, offset int) ([]domain.Order, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

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
	s.mu.Lock()
	defer s.mu.Unlock()

	orders, err := s.readAll()
	if err != nil {
		return nil, fmt.Errorf("ошибка при чтении заказов: %v", err)
	}

	sort.Slice(orders, func(i, j int) bool {
		return orders[i].UpdatedAt.After(orders[j].UpdatedAt)
	})

	return orders, nil
}
