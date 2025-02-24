package jsonstorage

import (
	"encoding/json"
	"os"

	"gitlab.ozon.dev/sadsnake2311/homework/internal/domain"
)
func (s *JSONOrderStorage) readAll() ([]domain.Order, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := os.ReadFile(s.filePath)
	if os.IsNotExist(err) {
		return []domain.Order{}, nil
	} else if err != nil {
		return nil, err
	}

	var orders []domain.Order
	if err := json.Unmarshal(data, &orders); err != nil {
		return nil, err
	}
	return orders, nil
}

func (s *JSONOrderStorage) writeAll(orders []domain.Order) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := json.MarshalIndent(orders, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.filePath, data, 0644) //0644 - read and write permission
}
