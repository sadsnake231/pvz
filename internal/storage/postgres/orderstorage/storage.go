package orderstorage

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/domain"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/storage/postgres/storageutils"
)

type OrderStorage struct {
	db *pgxpool.Pool
}

func NewOrderStorage(db *pgxpool.Pool) *OrderStorage {
	return &OrderStorage{db: db}
}

func (s *OrderStorage) SaveOrder(ctx context.Context, order domain.Order) error {
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	savePackagingQuery := `INSERT INTO packaging_types (id, packaging_price) VALUES ($1, $2)
                          ON CONFLICT (id) DO NOTHING`
	if _, err := tx.Exec(ctx, savePackagingQuery, string(order.Packaging), order.PackagePrice); err != nil {
		return err
	}

	saveOrderQuery := `INSERT INTO orders (
        order_id, recipient_id, expiry, stored_at, issued_at, refunded_at,
        base_price, weight, packaging
    ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	if _, err := tx.Exec(ctx, saveOrderQuery,
		order.ID,
		order.RecipientID,
		order.Expiry,
		order.StoredAt,
		order.IssuedAt,
		order.RefundedAt,
		order.BasePrice,
		order.Weight,
		order.Packaging,
	); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (s *OrderStorage) DeleteOrder(ctx context.Context, id string) error {
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	query := `DELETE FROM orders WHERE order_id = $1`
	if _, err := tx.Exec(ctx, query, id); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (s *OrderStorage) FindOrderByID(ctx context.Context, id string) (*domain.Order, error) {
	query := `SELECT 
			order_id, recipient_id, expiry,
			stored_at, issued_at, refunded_at,
			base_price, weight, packaging
		FROM orders WHERE order_id = $1`
	row := s.db.QueryRow(ctx, query, id)
	return storageutils.ScanOrder(row)
}
