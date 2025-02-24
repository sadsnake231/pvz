package jsonstorage

import (
	"sync"
	"fmt"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/domain"
)

type JSONOrderStorage struct {
	filePath string
	mu       sync.Mutex
}

func NewJSONOrderStorage(filePath string) *JSONOrderStorage {
	return &JSONOrderStorage{filePath: filePath}
}

func (s *JSONOrderStorage) SaveOrder(order domain.Order) error {


	orders, err := s.readAll()
	if err != nil {
		return err
	}

	orders = append(orders, order)
	return s.writeAll(orders)
}

func (s *JSONOrderStorage) FindOrderByID(id string) (int, *domain.Order, error) {
	orders, err := s.readAll()
	if err != nil {
		return -1, nil, err
	}

	for i, o := range orders {
		if o.ID == id {
			return i, &o, nil
		}
	}
	return -1, nil, nil
}

func (s *JSONOrderStorage) DeleteOrder(id string) (string, error) {

	orders, err := s.readAll()
	if err != nil {
		return "", fmt.Errorf("ошибка при чтении заказов: %v", err)
	}

	updatedOrders := make([]domain.Order, 0, len(orders))
	for _, order := range orders {
		if order.ID == id {
			continue
		}
		updatedOrders = append(updatedOrders, order)
	}

	if err := s.writeAll(updatedOrders); err != nil {
		return "", fmt.Errorf("ошибка при сохранении заказов: %v", err)
	}

	return id, nil
}

