package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/domain"
)

type UserOrderStorage struct {
	db *pgxpool.Pool
}

func NewUserOrderStorage(db *pgxpool.Pool) *UserOrderStorage {
	return &UserOrderStorage{db: db}
}

func (s *UserOrderStorage) IssueOrders(ctx context.Context, userID string, orderIDs []string) (domain.ProcessedOrders, error) {
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return domain.ProcessedOrders{}, fmt.Errorf("%w: %v", domain.ErrDatabase, err)
	}
	defer tx.Rollback(ctx)

	processed := make([]string, 0)
	now := time.Now().UTC()
	var returnErr error

	for _, id := range orderIDs {
		o, err := s.lockAndGetOrder(ctx, tx, id)
		if err != nil {
			if errors.Is(err, domain.ErrNotFoundOrder) {
				returnErr = fmt.Errorf("%w: %s", domain.ErrNotFoundOrder, id)
				continue
			}
			return domain.ProcessedOrders{}, fmt.Errorf("%w: %v", domain.ErrDatabase, err)
		}

		if err := validateOrderForIssue(o, userID, now); err != nil {
			returnErr = err
			break
		}

		if err := s.updateIssueTime(ctx, tx, id, now); err != nil {
			return domain.ProcessedOrders{}, fmt.Errorf("%w: %v", domain.ErrDatabase, err)
		}

		processed = append(processed, id)
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.ProcessedOrders{}, fmt.Errorf("%w: %v", domain.ErrDatabase, err)
	}

	return domain.ProcessedOrders{
		UserID:   userID,
		OrderIDs: processed,
		Failed:   getFailedOrders(orderIDs, processed),
		Error:    returnErr,
	}, nil
}

func (s *UserOrderStorage) RefundOrders(ctx context.Context, userID string, orderIDs []string) (domain.ProcessedOrders, error) {
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return domain.ProcessedOrders{}, fmt.Errorf("%w: %v", domain.ErrDatabase, err)
	}
	defer tx.Rollback(ctx)

	processed := make([]string, 0)
	now := time.Now().UTC()
	var returnErr error

	for _, id := range orderIDs {
		o, err := s.lockAndGetOrder(ctx, tx, id)
		if err != nil {
			if errors.Is(err, domain.ErrNotFoundOrder) {
				returnErr = fmt.Errorf("%w: %s", domain.ErrNotFoundOrder, id)
				continue
			}
			return domain.ProcessedOrders{}, fmt.Errorf("%w: %v", domain.ErrDatabase, err)
		}

		if err := validateOrderForRefund(o, userID, now); err != nil {
			returnErr = err
			break
		}

		if err := s.updateRefundTime(ctx, tx, id, now); err != nil {
			return domain.ProcessedOrders{}, fmt.Errorf("%w: %v", domain.ErrDatabase, err)
		}

		processed = append(processed, id)
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.ProcessedOrders{}, fmt.Errorf("%w: %v", domain.ErrDatabase, err)
	}

	return domain.ProcessedOrders{
		UserID:   userID,
		OrderIDs: processed,
		Failed:   getFailedOrders(orderIDs, processed),
		Error:    returnErr,
	}, nil
}

func (s *UserOrderStorage) lockAndGetOrder(ctx context.Context, tx pgx.Tx, id string) (*domain.Order, error) {
	query := `SELECT order_id, recipient_id, expiry, stored_at, issued_at, 
					 refunded_at, base_price, weight, packaging
	 		FROM orders WHERE order_id = $1 FOR UPDATE`
	row := tx.QueryRow(ctx, query, id)
	order, err := scanOrder(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFoundOrder
		}
		return nil, fmt.Errorf("%w: %v", domain.ErrDatabase, err)
	}
	return order, nil
}

func validateOrderForIssue(o *domain.Order, userID string, now time.Time) error {
	if o.RecipientID != userID {
		return &domain.ErrUserDoesntOwnOrder{OrderID: o.ID, UserID: userID}
	}

	if o.Status() != domain.StatusStored {
		return domain.ErrNotStoredOrder
	}

	if o.Expiry.Before(now) {
		return domain.ErrExpiredOrder
	}

	return nil
}

func validateOrderForRefund(o *domain.Order, userID string, now time.Time) error {
	if o.RecipientID != userID {
		return &domain.ErrUserDoesntOwnOrder{OrderID: o.ID, UserID: userID}
	}

	if o.Status() != domain.StatusIssued {
		return domain.ErrNotStoredOrder
	}

	if o.IssuedAt != nil && now.Sub(*o.IssuedAt) > 48*time.Hour {
		return domain.ErrRefundPeriodExpired
	}

	return nil
}

func (s *UserOrderStorage) updateIssueTime(ctx context.Context, tx pgx.Tx, id string, t time.Time) error {
	_, err := tx.Exec(ctx,
		"UPDATE orders SET issued_at = $1 WHERE order_id = $2",
		t, id,
	)
	if err != nil {
		return fmt.Errorf("%w: %v", domain.ErrDatabase, err)
	}
	return nil
}

func (s *UserOrderStorage) updateRefundTime(ctx context.Context, tx pgx.Tx, id string, t time.Time) error {
	_, err := tx.Exec(ctx,
		"UPDATE orders SET refunded_at = $1 WHERE order_id = $2",
		t, id,
	)
	if err != nil {
		return fmt.Errorf("%w: %v", domain.ErrDatabase, err)
	}
	return nil
}

func getFailedOrders(orderIDs, processed []string) []string {
	processedMap := make(map[string]struct{}, len(processed))
	for _, id := range processed {
		processedMap[id] = struct{}{}
	}

	var failed []string
	for _, id := range orderIDs {
		if _, found := processedMap[id]; !found {
			failed = append(failed, id)
		}
	}
	return failed
}
