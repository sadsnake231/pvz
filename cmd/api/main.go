package main

import (
	"log"

	"gitlab.ozon.dev/sadsnake2311/homework/internal/api"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/config"
	database "gitlab.ozon.dev/sadsnake2311/homework/internal/db"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/repository/authrepo"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/repository/orderrepo"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/repository/reportrepo"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/repository/userorderrepo"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/router"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/service"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/storage/postgres/authstorage"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/storage/postgres/orderstorage"
	reportorder "gitlab.ozon.dev/sadsnake2311/homework/internal/storage/postgres/reportorderstorage"
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
		logger.Error("Не удалось подключиться к базе данных", zap.Error(err))
	}
	defer db.Close()

	orderStorage := orderstorage.NewOrderStorage(db)
	userOrderStorage := userorder.NewUserOrderStorage(db)
	reportOrderStorage := reportorder.NewReportOrderStorage(db)
	authStorage := authstorage.NewAuthStorage(db)

	orderRepo := orderrepo.NewOrderRepository(orderStorage, logger)
	userRepo := userorderrepo.NewUserOrderRepository(userOrderStorage, logger)
	reportRepo := reportrepo.NewReportRepository(reportOrderStorage, logger)
	authRepo := authrepo.NewAuthRepository(authStorage, logger)

	orderService := service.NewOrderService(orderRepo, userRepo, reportRepo)
	authService := service.NewAuthService(authRepo)

	apiHandler := api.NewAPIHandler(orderService)
	authHandler := api.NewAuthHandler(authService, logger)

	router := router.SetupRouter(apiHandler, authHandler, logger)

	if err := router.Run(cfg.HTTPPort); err != nil {
		logger.Error("Сервер не запустился", zap.Error(err))
	}

}
