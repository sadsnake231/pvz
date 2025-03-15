package reportorderstorage

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/domain"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/storage/postgres/storageutils"
)

type ReportOrderStorage struct {
	db *pgxpool.Pool
}

func NewReportOrderStorage(db *pgxpool.Pool) *ReportOrderStorage {
	return &ReportOrderStorage{db: db}
}

func (s *ReportOrderStorage) GetUserOrders(
	ctx context.Context,
	userID string,
	limit int,
	cursor *int,
	status string,
) ([]domain.Order, string, error) {
	query := `SELECT 
	order_id, recipient_id, expiry, 
	stored_at, issued_at, refunded_at, 
	base_price, weight, packaging
	FROM orders
	WHERE 
    recipient_id = $1 AND
    ($2::INT IS NULL OR id < $2) AND
    ($4 = '' OR (
            $4 = 'stored' AND 
            stored_at IS NOT NULL AND
            issued_at IS NULL AND
            refunded_at IS NULL
        )
    )
	ORDER BY id DESC
	LIMIT $3
    `

	rows, err := s.db.Query(ctx, query, userID, cursor, limit, status)
	if err != nil {
		return nil, "", fmt.Errorf("ошибка запроса: %w", err)
	}
	defer rows.Close()

	orders, err := scanOrders(rows)
	if err != nil {
		return nil, "", fmt.Errorf("ошибка скана: %w", err)
	}

	var nextCursor string
	if len(orders) > 0 {
		query = `SELECT id FROM orders WHERE order_id = $1`
		row := s.db.QueryRow(ctx, query, orders[len(orders)-1].ID)

		var nextID int
		if err := row.Scan(&nextID); err != nil {
			return nil, "", fmt.Errorf("ошибка получения курсора: %w", err)
		}
		nextCursor = strconv.Itoa(nextID)
	}

	return orders, nextCursor, nil
}

func (s *ReportOrderStorage) GetRefundedOrders(
	ctx context.Context,
	limit int,
	cursor *int,
) ([]domain.Order, string, error) {
	query := `
		SELECT 
		order_id, recipient_id, expiry, 
		stored_at, issued_at, refunded_at, 
		base_price, weight, packaging
		FROM orders
		WHERE 
			refunded_at IS NOT NULL AND
			($1::INT IS NULL OR id < $1)
		ORDER BY id DESC
		LIMIT $2
	`

	rows, err := s.db.Query(ctx, query, cursor, limit)
	if err != nil {
		return nil, "", fmt.Errorf("ошибка запроса: %w", err)
	}
	defer rows.Close()

	orders, err := scanOrders(rows)
	if err != nil {
		return nil, "", fmt.Errorf("ошибка скана: %w", err)
	}

	var nextCursor string
	if len(orders) > 0 {
		query = `SELECT id FROM orders WHERE order_id = $1`
		row := s.db.QueryRow(ctx, query, orders[len(orders)-1].ID)

		var nextID int
		if err := row.Scan(&nextID); err != nil {
			return nil, "", fmt.Errorf("ошибка получения следующего курсора: %w", err)
		}
		nextCursor = strconv.Itoa(nextID)
	}

	return orders, nextCursor, nil
}

func (s *ReportOrderStorage) GetOrderHistory(
	ctx context.Context,
	limit int,
	lastUpdatedCursor time.Time,
	idCursor int,
) ([]domain.Order, string, error) {
	query := `
        SELECT 
    	order_id, recipient_id, expiry, 
    	stored_at, issued_at, refunded_at, 
    	base_price, weight, packaging
		FROM orders
		WHERE 
    	($1::timestamp = '0001-01-01' AND $2 = 0) OR  
    	(
        	GREATEST(
            	COALESCE(stored_at, '0001-01-01'::timestamp),
            	COALESCE(issued_at, '0001-01-01'::timestamp),
            	COALESCE(refunded_at, '0001-01-01'::timestamp)
        ) < $1 OR
        (
            GREATEST(
                COALESCE(stored_at, '0001-01-01'::timestamp),
                COALESCE(issued_at, '0001-01-01'::timestamp),
                COALESCE(refunded_at, '0001-01-01'::timestamp)
            ) = $1 AND
            id < $2
        	)
    	)
		ORDER BY 
    	GREATEST(
        	COALESCE(stored_at, '0001-01-01'::timestamp),
        	COALESCE(issued_at, '0001-01-01'::timestamp),
        	COALESCE(refunded_at, '0001-01-01'::timestamp)
    	) DESC, 
    	id DESC
		LIMIT $3
    	`

	rows, err := s.db.Query(
		ctx,
		query,
		lastUpdatedCursor,
		idCursor,
		limit,
	)
	if err != nil {
		return nil, "", fmt.Errorf("ошибка запроса: %w", err)
	}
	defer rows.Close()

	orders, err := scanOrders(rows)
	if err != nil {
		return nil, "", fmt.Errorf("ошибка скана: %w", err)
	}

	var nextCursor string
	if len(orders) > 0 {
		lastOrder := orders[len(orders)-1]

		var nextID int
		query = `SELECT id FROM orders WHERE order_id = $1`
		row := s.db.QueryRow(ctx, query, lastOrder.ID)
		if err := row.Scan(&nextID); err != nil {
			return nil, "", fmt.Errorf("ошибка получения следующего курсора: %w", err)
		}

		nextCursor = fmt.Sprintf(
			"%s,%d",
			lastOrder.LastUpdated().Format(time.RFC3339Nano),
			nextID,
		)
	}

	return orders, nextCursor, nil
}

func (s *ReportOrderStorage) queryOrders(ctx context.Context, query string, args ...any) ([]domain.Order, error) {
	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("ошибка запроса: %w", err)
	}
	defer rows.Close()

	return scanOrders(rows)
}

func scanOrders(rows pgx.Rows) ([]domain.Order, error) {
	var orders []domain.Order
	for rows.Next() {
		o, err := storageutils.ScanOrder(rows)
		if err != nil {
			return nil, err
		}
		orders = append(orders, *o)
	}
	return orders, nil
}
