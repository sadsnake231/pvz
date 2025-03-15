package authstorage

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/domain"
)

type AuthStorage struct {
	db *pgxpool.Pool
}

func NewAuthStorage(db *pgxpool.Pool) *AuthStorage {
	return &AuthStorage{db: db}
}

func (s *AuthStorage) CreateUser(ctx context.Context, user *domain.User) error {
	query := `
        INSERT INTO users (email, password) 
        VALUES ($1, $2)`

	_, err := s.db.Exec(ctx, query, user.Email, user.Password)
	return err
}

func (s *AuthStorage) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `
        SELECT email, password 
        FROM users 
        WHERE email = $1`

	var user domain.User
	err := s.db.QueryRow(ctx, query, email).Scan(
		&user.Email,
		&user.Password,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrInvalidCredentials
		}
		return nil, err
	}
	return &user, nil
}
