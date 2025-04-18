package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"time"

	"gitlab.ozon.dev/sadsnake2311/homework/internal/api"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/audit"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/cache"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/config"
	database "gitlab.ozon.dev/sadsnake2311/homework/internal/db"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/kafka"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/metrics"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/middleware"
	auditrepo "gitlab.ozon.dev/sadsnake2311/homework/internal/repository/auditlogrepo"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/repository/authrepo"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/repository/orderrepo"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/repository/reportrepo"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/repository/userorderrepo"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/router"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/service"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/storage/postgres/auditlogstorage"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/storage/postgres/authstorage"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/storage/postgres/orderstorage"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/storage/postgres/reportorderstorage"
	userorder "gitlab.ozon.dev/sadsnake2311/homework/internal/storage/postgres/userorderstorage"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/tracing"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/transport/grpc"
	"go.uber.org/zap"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	cfg := config.Load()

	baseLogger, err := zap.NewProduction()
	logger := baseLogger.Sugar()
	if err != nil {
		log.Fatalf("failed to init logger: %v", err)
	}
	defer logger.Sync()

	tp, err := tracing.InitTracer(ctx, cfg.JaegerServiceName, cfg.JaegerURL)
	if err != nil {
		logger.Fatalf("failed to init tracer: %v", err)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := tp.Shutdown(shutdownCtx); err != nil {
			logger.Errorf("error shutting down tracer: %v", err)
		}
	}()

	db, err := database.NewDatabase(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Fatal("failed to init database", zap.Error(err))
	}
	defer db.Close()

	redisClient := cache.GetRedisClient(cfg.CacheURL, cfg.CachePassword)

	err = metrics.RegisterMetrics()
	if err != nil {
		logger.Error("failed to register metric", zap.Error(err))
	}

	orderStorage := orderstorage.NewOrderStorage(db)
	userOrderStorage := userorder.NewUserOrderStorage(db)
	reportStorage := reportorderstorage.NewReportOrderStorage(db)
	authStorage := authstorage.NewAuthStorage(db)
	auditStorage := auditlogstorage.NewAuditStorage(db)

	orderRepo := orderrepo.NewOrderRepository(orderStorage, logger)
	userRepo := userorderrepo.NewUserOrderRepository(userOrderStorage, logger)
	reportRepo := reportrepo.NewReportRepository(reportStorage, logger)

	authRepo := authrepo.NewAuthRepository(authStorage, logger)
	auditRepo := auditrepo.NewAuditRepository(auditStorage, logger)

	cache := cache.NewRedisCache(redisClient, reportRepo)

	orderService := service.NewOrderService(orderRepo, userRepo, reportRepo, cache, logger)
	authService := service.NewAuthService(authRepo)
	auditService := service.NewAuditService(auditRepo)

	dbPool := audit.NewWorkerPool(logger)
	stdoutPool := audit.NewWorkerPool(logger)
	filterFunc := audit.NewFilterFunc(cfg.AuditFilter)
	auditPipeline := audit.NewPipeline(auditService, logger, dbPool, stdoutPool)

	apiHandler := api.NewAPIHandler(orderService, auditPipeline)
	authHandler := api.NewAuthHandler(authService, logger)

	kafkaProducer, err := kafka.NewProducer(cfg.KafkaBrokers, logger)
	if err != nil {
		logger.Fatalw("failed to init Kafka Producer", "error", err)
	}

	outboxWorker := audit.NewOutboxWorker(auditService, kafkaProducer, logger)
	go outboxWorker.Run(ctx)

	auditPipeline.StartWorkers(ctx, filterFunc)

	orderService.InitCache(ctx)
	go orderService.CacheRefresh(ctx)

	router := router.SetupRouter(apiHandler, authHandler, logger, auditPipeline)
	router.Use(middleware.AuditMiddleware(auditPipeline))

	go func() {
		router.Run(cfg.HTTPPort)
	}()

	grpcServer := grpc.NewServer(
		orderService,
		authService,
		auditPipeline,
		logger,
	)

	go func() {
		logger.Info("starting gRPC server", zap.String("port", cfg.GRPCPort))
		if err := grpcServer.Run(cfg.GRPCPort); err != nil {
			logger.Error("failed to start gRPC server", zap.Error(err))
		}
	}()

	<-ctx.Done()
	grpcServer.Stop()

	logger.Info("waiting for logger to shut down...")

	logger.Info("shutdown complete")
}
