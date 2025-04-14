package handler

import (
	"context"
	"errors"

	"gitlab.ozon.dev/sadsnake2311/homework/internal/domain"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/service"
	grpcapi "gitlab.ozon.dev/sadsnake2311/homework/internal/transport/grpc/gen"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type AuthHandler struct {
	grpcapi.UnimplementedAuthHandlerServer
	service service.AuthService
	logger  *zap.SugaredLogger
}

func NewAuthHandler(service service.AuthService, logger *zap.SugaredLogger) *AuthHandler {
	return &AuthHandler{service: service, logger: logger}
}

func (h *AuthHandler) Signup(ctx context.Context, req *grpcapi.SignupRequest) (*grpcapi.SignupResponse, error) {
	if req.GetEmail() == "" || req.GetPassword() == "" {
		return nil, status.Error(codes.InvalidArgument, "нужно указать email и пароль")
	}

	user := domain.User{
		Email:    req.GetEmail(),
		Password: req.GetPassword(),
	}

	if err := h.service.Register(ctx, &user); err != nil {
		return nil, convertAuthError(err)
	}

	return &grpcapi.SignupResponse{Message: "пользователь зарегистрирован"}, nil
}

func (h *AuthHandler) Login(ctx context.Context, req *grpcapi.LoginRequest) (*grpcapi.LoginResponse, error) {
	user := domain.User{
		Email:    req.GetEmail(),
		Password: req.GetPassword(),
	}

	if err := h.service.Login(ctx, &user); err != nil {
		return nil, convertAuthError(err)
	}

	token, err := h.service.GenerateToken(user.Email)
	if err != nil {
		return nil, status.Error(codes.Internal, "ошибка генерации токена")
	}

	return &grpcapi.LoginResponse{
		Message: "вход успешен",
		Token:   token,
	}, nil
}

func convertAuthError(err error) error {
	switch {
	case errors.Is(err, domain.ErrUserAlreadyExists):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, domain.ErrInvalidCredentials):
		return status.Error(codes.Unauthenticated, err.Error())
	case errors.Is(err, domain.ErrDatabase), errors.Is(err, domain.ErrHashPassword):
		return status.Error(codes.Internal, err.Error())
	default:
		return status.Error(codes.InvalidArgument, err.Error())
	}
}
