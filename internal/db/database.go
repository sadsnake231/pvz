package database

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

func NewDatabase(connStr string) (*pgxpool.Pool, error) {
	db, err := pgxpool.New(context.Background(), connStr)
	if err != nil {
		return nil, err
	}

	return db, err
}
