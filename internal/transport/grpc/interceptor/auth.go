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
	if shouldSkipAuth(info.FullMethod) {
		return handler(ctx, req)
	}

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "отсутствуют метаданные")
	}

	tokenString, err := extractTokenFromMetadata(md)
	if err != nil {
		return nil, err
	}

	token, err := jwt.Parse(tokenString, jwtKeyFunc)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "ошибка проверки токена: %v", err)
	}
	if !token.Valid {
		return nil, status.Error(codes.Unauthenticated, "неверный токен")
	}

	return handler(ctx, req)
}

func shouldSkipAuth(fullMethod string) bool {
	skipMethods := map[string]bool{
		"/transport.grpc.auth.AuthHandler/Login":  true,
		"/transport.grpc.auth.AuthHandler/Signup": true,
	}
	return skipMethods[fullMethod]
}

func extractTokenFromMetadata(md metadata.MD) (string, error) {
	authHeader := md.Get("authorization")
	if len(authHeader) == 0 {
		return "", status.Error(codes.Unauthenticated, "отсутствует заголовок авторизации")
	}

	return strings.TrimPrefix(authHeader[0], "Bearer "), nil
}

func jwtKeyFunc(token *jwt.Token) (any, error) {
	return []byte(domain.JwtSecret), nil
}
