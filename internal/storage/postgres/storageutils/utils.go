package storageutils

import (
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/domain"
)

func ScanOrder(row pgx.Row) (*domain.Order, error) {
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
