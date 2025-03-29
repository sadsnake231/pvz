package main

import (
	"context"
	"log"
	"os"
	"os/signal"

	"gitlab.ozon.dev/sadsnake2311/homework/internal/api"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/audit"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/cache"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/config"
	database "gitlab.ozon.dev/sadsnake2311/homework/internal/db"
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
	"go.uber.org/zap"
)

func main() {
	cfg := config.Load()

	baseLogger, err := zap.NewProduction()
	logger := baseLogger.Sugar()
	if err != nil {
		log.Fatalf("Ошибка старта логгера: %v", err)
	}
	defer logger.Sync()

	db, err := database.NewDatabase(cfg.DatabaseURL)
	if err != nil {
		logger.Fatal("Не удалось подключиться к базе данных", zap.Error(err))
	}
	defer db.Close()

	redisClient := cache.GetRedisClient("localhost:6379", "")
	cache := cache.NewRedisCache(redisClient)

	metrics.RegisterMetrics()
	metrics.StartMetricsServer()

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

	orderService := service.NewOrderService(orderRepo, userRepo, reportRepo, cache, logger)
	authService := service.NewAuthService(authRepo)
	auditService := service.NewAuditService(auditRepo)

	dbPool := audit.NewWorkerPool(logger)
	stdoutPool := audit.NewWorkerPool(logger)
	filterFunc := audit.NewFilterFunc(cfg.AuditFilter)
	auditPipeline := audit.NewPipeline(auditService, logger, dbPool, stdoutPool)

	apiHandler := api.NewAPIHandler(orderService, auditPipeline)
	authHandler := api.NewAuthHandler(authService, logger)
	auditHandler := api.NewAuditHandler(auditService)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	auditPipeline.StartWorkers(ctx, filterFunc)

	orderService.InitCache(ctx)
	go orderService.CacheRefresh(ctx)

	router := router.SetupRouter(apiHandler, authHandler, auditHandler, logger, auditPipeline)
	router.Use(middleware.AuditMiddleware(auditPipeline))

	serverErr := make(chan error, 1)
	go func() {
		serverErr <- router.Run(cfg.HTTPPort)
	}()

	select {
	case <-ctx.Done():
		logger.Info("Получен сигнал завершения, выключаем сервер...")
	case err := <-serverErr:
		logger.Fatal("Ошибка работы сервера", zap.Error(err))
	}

	stop()
	logger.Info("Ожидание завершения работы логгера...")

	logger.Info("Выключение завершено")
}
