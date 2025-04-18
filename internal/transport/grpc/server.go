package grpc

import (
	"net"

	"gitlab.ozon.dev/sadsnake2311/homework/internal/audit"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/service"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/transport/grpc/gen/auth"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/transport/grpc/gen/order"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/transport/grpc/handler"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/transport/grpc/interceptor"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

type Server struct {
	server *grpc.Server
	logger *zap.SugaredLogger
}

func NewServer(
	orderService service.OrderService,
	authService service.AuthService,
	auditPipeline *audit.Pipeline,
	logger *zap.SugaredLogger,
) *Server {
	interceptors := grpc.ChainUnaryInterceptor(
		interceptor.MetricsInterceptor,
		interceptor.AuthInterceptor,
		interceptor.AuditInterceptor(auditPipeline),
		otelgrpc.UnaryServerInterceptor(),
	)

	grpcServer := grpc.NewServer(interceptors)

	orderHandler := handler.NewOrderHandler(orderService, auditPipeline)
	authHandler := handler.NewAuthHandler(authService, logger)

	order.RegisterOrderHandlerServer(grpcServer, orderHandler)
	auth.RegisterAuthHandlerServer(grpcServer, authHandler)

	return &Server{
		server: grpcServer,
		logger: logger,
	}
}

func (s *Server) Run(port string) error {
	lis, err := net.Listen("tcp", port)
	if err != nil {
		return err
	}
	s.logger.Info("gRPC server started")
	return s.server.Serve(lis)
}

func (s *Server) Stop() {
	s.server.GracefulStop()
}
