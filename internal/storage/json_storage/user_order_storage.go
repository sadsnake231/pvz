package jsonstorage

import (
	"fmt"
	"time"

	"gitlab.ozon.dev/sadsnake2311/homework/internal/domain"
)

func (s *JSONOrderStorage) IssueOrders(userID string, orderIDs []string) (string, []string, error) {
	orders, err := s.readAll()
	if err != nil {
		return "", nil, fmt.Errorf("ошибка при чтении заказов: %v", err)
	}

	issuedOrderIDs, returnErr := s.processOrdersForIssue(userID, orderIDs, orders)

	if err := s.writeAll(orders); err != nil {
		return "", nil, fmt.Errorf("ошибка при сохранении заказов: %v", err)
	}

	if len(issuedOrderIDs) == 0 && returnErr == nil {
		returnErr = domain.ErrUserNoOrders
	}

	return userID, issuedOrderIDs, returnErr
}

func (s *JSONOrderStorage) processOrdersForIssue(userID string, orderIDs []string, orders []domain.Order) ([]string, error) {
	issuedOrderIDs := make([]string, 0)
	var returnErr error

	for _, orderID := range orderIDs {
		index, order, err := s.FindOrderByID(orderID)
		if err != nil {
			return nil, err
		}

		if err := s.validateOrderForIssue(userID, order); err != nil {
			returnErr = err
			break
		}

		orders[index].Status = domain.StatusIssued
		orders[index].UpdatedAt = time.Now()
		issuedOrderIDs = append(issuedOrderIDs, orderID)
	}

	return issuedOrderIDs, returnErr
}

func (s *JSONOrderStorage) validateOrderForIssue(userID string, order *domain.Order) error {
	if order.RecipientID != userID {
		return &domain.ErrUserDoesntOwnOrder{OrderID: order.ID, UserID: userID}
	}

	if order.Status != domain.StatusStored {
		return domain.ErrNotStoredOrder
	}

	if order.Expiry.Before(time.Now()) {
		return domain.ErrExpiredOrder
	}

	return nil
}

func (s *JSONOrderStorage) RefundOrders(userID string, orderIDs []string) (string, []string, error) {
	orders, err := s.readAll()
	if err != nil {
		return "", nil, fmt.Errorf("ошибка при чтении заказов: %v", err)
	}

	refundedOrderIDs, returnErr := s.processOrdersForRefund(userID, orderIDs, orders)

	if err := s.writeAll(orders); err != nil {
		return "", nil, fmt.Errorf("ошибка при сохранении заказов: %v", err)
	}

	if len(refundedOrderIDs) == 0 && returnErr == nil {
		returnErr = domain.ErrUserNoOrders
	}

	return userID, refundedOrderIDs, returnErr
}

func (s *JSONOrderStorage) processOrdersForRefund(userID string, orderIDs []string, orders []domain.Order) ([]string, error) {
	refundedOrderIDs := make([]string, 0)
	var returnErr error

	for _, orderID := range orderIDs {
		index, order, err := s.FindOrderByID(orderID)
		if err != nil {
			return nil, err
		}

		if err := s.validateOrderForRefund(userID, order); err != nil {
			returnErr = err
			break
		}

		orders[index].Status = domain.StatusRefunded
		orders[index].UpdatedAt = time.Now()
		refundedOrderIDs = append(refundedOrderIDs, orderID)
	}

	return refundedOrderIDs, returnErr
}

func (s *JSONOrderStorage) validateOrderForRefund(userID string, order *domain.Order) error {
	if order.RecipientID != userID {
		return &domain.ErrUserDoesntOwnOrder{OrderID: order.ID, UserID: userID}
	}

	if order.Status != domain.StatusIssued {
		return domain.ErrUserNoOrders
	}

	if time.Since(order.UpdatedAt) > 48*time.Hour {
		return domain.ErrRefundPeriodExpired
	}

	return nil
}
