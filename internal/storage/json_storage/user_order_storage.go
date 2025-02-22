package jsonstorage

import (
	"fmt"
	"time"

	"gitlab.ozon.dev/sadsnake2311/homework/hw-1/internal/domain"
)

func (s *JSONOrderStorage) IssueOrders(userID string, orderIDs []string) (string, []string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	issuedOrderIDs := make([]string, 0)

	orders, err := s.readAll()
	if err != nil {
		return "", nil, fmt.Errorf("ошибка при чтении заказов: %v", err)
	}

	var returnErr error

	for _, orderID := range orderIDs {
		index, order, err := s.FindOrderByID(orderID)
		if err != nil {
			return "", nil, err
		}

		if order.RecipientID != userID {
			returnErr = &domain.ErrUserDoesntOwnOrder{OrderID: orderID, UserID: userID}
			break
		}

		if order.Status != domain.StatusStored {
			returnErr = domain.ErrNotStoredOrder
			break
		}

		if order.Expiry.Before(time.Now()) {
			returnErr = domain.ErrExpiredOrder
			break
		}
		orders[index].Status = domain.StatusIssued
		orders[index].UpdatedAt = time.Now()

		issuedOrderIDs = append(issuedOrderIDs, orderID)
	}

	if err := s.writeAll(orders); err != nil {
		return "", nil, fmt.Errorf("ошибка при сохранении заказов: %v", err)
	}

	if len(issuedOrderIDs) == 0 && returnErr == nil{
		returnErr = domain.ErrUserNoOrders
	}
	return userID, issuedOrderIDs, returnErr
}

func (s *JSONOrderStorage) RefundOrders(userID string, orderIDs []string) (string, []string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	refundedOrderIDs := make([]string, 0)

	orders, err := s.readAll()
	if err != nil {
		return "", nil, fmt.Errorf("ошибка при чтении заказов: %v", err)
	}

	var returnErr error

	for _, orderID := range orderIDs {
		index, order, err := s.FindOrderByID(orderID)
		if err != nil {
			return "", nil, err
		}

		if order.RecipientID != userID {
			returnErr = &domain.ErrUserDoesntOwnOrder{OrderID: orderID, UserID: userID}
			break
		}

		if order.Status != domain.StatusIssued {
			returnErr = domain.ErrUserNoOrders
			break
		}

		if time.Since(order.UpdatedAt) > 48*time.Hour {
			returnErr = domain.ErrRefundPeriodExpired
			break
		}


		orders[index].Status = domain.StatusRefunded
		orders[index].UpdatedAt = time.Now()

		refundedOrderIDs = append(refundedOrderIDs, orderID)
	}

	if err := s.writeAll(orders); err != nil {
		return "", nil, fmt.Errorf("ошибка при сохранении заказов: %v", err)
	}

	if len(refundedOrderIDs) == 0 && returnErr == nil {
		returnErr = domain.ErrUserNoOrders
	}
	return userID, refundedOrderIDs, returnErr
}
