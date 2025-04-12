package interceptor

import (
	"context"
	"time"

	"gitlab.ozon.dev/sadsnake2311/homework/internal/metrics"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

func MetricsInterceptor(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	metrics.RequestsInFlight.WithLabelValues(info.FullMethod).Inc()
	defer metrics.RequestsInFlight.WithLabelValues(info.FullMethod).Dec()

	start := time.Now()

	resp, err := handler(ctx, req)
	duration := time.Since(start).Seconds()
	statusCode := status.Code(err).String()

	metrics.APIResponseTime.WithLabelValues(
		info.FullMethod,
		"unary",
		statusCode,
	).Observe(duration)

	metrics.RequestCount.WithLabelValues(
		info.FullMethod,
		statusCode,
	).Inc()

	return resp, err
}
