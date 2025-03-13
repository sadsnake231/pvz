package domain

import (
	"errors"
	"time"
)

const (
	JwtSecret       = "OZONROUTE256GOJUNIOR"
	TokenExpiration = 24 * time.Hour
)

var (
	ErrUserAlreadyExists  = errors.New("пользователь с таким email уже зарегистрирован")
	ErrInvalidCredentials = errors.New("email или пароль неверны")
	ErrEmptyPassword      = errors.New("пароль не может быть пустым")
	ErrHashPassword       = errors.New("ошибка хеширования пароля")
	ErrTokenGeneration    = errors.New("ошибка генерации токена")
)

type User struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}
