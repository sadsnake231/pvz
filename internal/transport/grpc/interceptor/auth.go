package interceptor

import (
	"context"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/domain"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func AuthInterceptor(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	if info.FullMethod == "/transport.grpc.AuthHandler/Login" ||
		info.FullMethod == "/transport.grpc.AuthHandler/Signup" {
		return handler(ctx, req)
	}

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "отсутствуют метаданные")
	}

	authHeader := md.Get("authorization")
	if len(authHeader) == 0 {
		return nil, status.Error(codes.Unauthenticated, "отсутствует заголовок авторизации")
	}

	tokenString := strings.TrimPrefix(authHeader[0], "Bearer ")
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
		return []byte(domain.JwtSecret), nil
	})

	if err != nil || !token.Valid {
		return nil, status.Error(codes.Unauthenticated, "неверный токен")
	}

	return handler(ctx, req)
}
