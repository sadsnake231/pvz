package repository

import (
	"context"
	"errors"

	"gitlab.ozon.dev/sadsnake2311/homework/internal/domain"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/storage"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

type AuthRepository struct {
	storage storage.AuthStorage
	logger  *zap.Logger
}

func NewAuthRepository(storage storage.AuthStorage, logger *zap.Logger) *AuthRepository {
	return &AuthRepository{storage: storage, logger: logger}
}

func (r *AuthRepository) Register(ctx context.Context, user *domain.User) error {
	if user.Password == "" {
		return domain.ErrEmptyPassword
	}

	exists, err := r.storage.GetUserByEmail(ctx, user.Email)
	if err != nil && !errors.Is(err, domain.ErrInvalidCredentials) {
		r.logger.Error("ошибка при поиске пользователя в базе данных", zap.Error(err))
		return domain.ErrDatabase
	}

	if exists != nil {
		return domain.ErrUserAlreadyExists
	}

	user.Password, err = hashPassword(user.Password)
	if err != nil {
		r.logger.Error("ошибка криптографии", zap.Error(err))
		return domain.ErrHashPassword
	}

	err = r.storage.CreateUser(ctx, user)
	if err != nil {
		r.logger.Error("ошибка сохранения пользователя в базе данных", zap.Error(err))
		return domain.ErrDatabase
	}
	return nil
}

func (r *AuthRepository) Login(ctx context.Context, email, password string) error {
	user, err := r.storage.GetUserByEmail(ctx, email)
	if err != nil && !errors.Is(err, domain.ErrInvalidCredentials) {
		r.logger.Error("ошибка при поиске пользователя в базе данных", zap.Error(err))
		return domain.ErrDatabase
	}

	if !compareHashAndPassword(password, user.Password) {
		return domain.ErrInvalidCredentials
	}

	return nil
}

func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

func compareHashAndPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
