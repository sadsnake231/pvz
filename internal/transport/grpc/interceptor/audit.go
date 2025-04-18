package interceptor

import (
	"context"

	"gitlab.ozon.dev/sadsnake2311/homework/internal/audit"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/domain"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/metrics"
	"google.golang.org/grpc"
)

func AuditInterceptor(p *audit.Pipeline) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		metrics.IncHTTPRequest(info.FullMethod)

		p.SendEvent(domain.EventAPIRequest, map[string]any{
			"method": info.FullMethod,
		})

		resp, err := handler(ctx, req)

		status := "OK"
		if err != nil {
			status = "Error"
		}

		metrics.IncHTTPResponse(status, info.FullMethod)

		p.SendEvent(domain.EventAPIResponse, map[string]any{
			"method": info.FullMethod,
			"status": status,
		})

		return resp, err
	}
}
