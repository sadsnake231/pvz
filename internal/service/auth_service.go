package service

import (
	"context"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/domain"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/repository"
)

type AuthService interface {
	Register(ctx context.Context, user *domain.User) error
	Login(ctx context.Context, user *domain.User) error
	GenerateToken(email string) (string, error)
}

type authService struct {
	repo *repository.AuthRepository
}

func NewAuthService(repo *repository.AuthRepository) AuthService {
	return &authService{repo: repo}
}

func (s *authService) Register(ctx context.Context, user *domain.User) error {
	return s.repo.Register(ctx, user)
}

func (s *authService) Login(ctx context.Context, user *domain.User) error {
	return s.repo.Login(ctx, user.Email, user.Password)
}

func (s *authService) GenerateToken(email string) (string, error) {
	claims := jwt.MapClaims{
		"email": email,
		"exp":   time.Now().Add(domain.TokenExpiration).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(domain.JwtSecret))
}
