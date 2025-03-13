package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/domain"
)

type OrderStorage struct {
	db *pgxpool.Pool
}

func NewOrderStorage(db *pgxpool.Pool) *OrderStorage {
	return &OrderStorage{db: db}
}

func (s *OrderStorage) SaveOrder(ctx context.Context, order domain.Order) error {
	savePackagingQuery := `INSERT INTO packaging_types (id, packaging_price) VALUES ($1, $2)
	                       ON CONFLICT (id) DO NOTHING`
	_, err := s.db.Exec(ctx, savePackagingQuery, string(order.Packaging), order.PackagePrice)
	if err != nil {
		return fmt.Errorf("failed to save packaging: %w", err)
	}

	saveOrderQuery := `INSERT INTO orders (
		order_id, recipient_id, expiry, stored_at, issued_at, refunded_at,
		base_price, weight, packaging
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	_, err = s.db.Exec(ctx, saveOrderQuery,
		order.ID,
		order.RecipientID,
		order.Expiry,
		order.StoredAt,
		order.IssuedAt,
		order.RefundedAt,
		order.BasePrice,
		order.Weight,
		order.Packaging,
	)

	return err
}

func (s *OrderStorage) FindOrderByID(ctx context.Context, id string) (*domain.Order, error) {
	query := `SELECT * FROM orders WHERE order_id = $1`
	row := s.db.QueryRow(ctx, query, id)
	return scanOrder(row)
}

func (s *OrderStorage) DeleteOrder(ctx context.Context, id string) error {
	query := `DELETE FROM orders WHERE order_id = $1`
	_, err := s.db.Exec(ctx, query, id)
	if err != nil {
		return err
	}

	return nil
}

func scanOrder(row pgx.Row) (*domain.Order, error) {
	var o domain.Order
	err := row.Scan(
		&o.ID,
		&o.RecipientID,
		&o.Expiry,
		&o.StoredAt,
		&o.IssuedAt,
		&o.RefundedAt,
		&o.BasePrice,
		&o.Weight,
		&o.Packaging,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFoundOrder
	}
	if err != nil {
		return nil, fmt.Errorf("не смог отсканить заказ: %w", err)
	}

	return &o, nil
}
