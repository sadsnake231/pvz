package interceptor

import (
	"context"
	"time"

	"gitlab.ozon.dev/sadsnake2311/homework/internal/metrics"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

func MetricsInterceptor(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	metrics.IncRequestsInFlight(info.FullMethod)
	defer metrics.DecRequestsInFlight(info.FullMethod)

	start := time.Now()
	resp, err := handler(ctx, req)
	duration := time.Since(start)
	statusCode := status.Code(err).String()

	metrics.ObserveAPIResponseTime(info.FullMethod, statusCode, duration)
	metrics.IncRequestCount(info.FullMethod, statusCode)

	return resp, err
}
