package main

import (
	"log"

	"gitlab.ozon.dev/sadsnake2311/homework/internal/api"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/config"
	database "gitlab.ozon.dev/sadsnake2311/homework/internal/db"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/repository"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/router"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/service"
	postgres "gitlab.ozon.dev/sadsnake2311/homework/internal/storage/postgres_storage"
	"go.uber.org/zap"
)

func main() {
	cfg := config.Load()

	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatalf("Ошибка старта логгера")
	}
	defer logger.Sync()

	db, err := database.NewDatabase(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	orderStorage := postgres.NewOrderStorage(db)
	userOrderStorage := postgres.NewUserOrderStorage(db)
	reportOrderStorage := postgres.NewReportOrderStorage(db)
	authStorage := postgres.NewAuthStorage(db)

	orderRepo := repository.NewOrderRepository(orderStorage, logger)
	userRepo := repository.NewUserOrderRepository(userOrderStorage, logger)
	reportRepo := repository.NewReportRepository(reportOrderStorage, logger)
	authRepo := repository.NewAuthRepository(authStorage, logger)

	storageService := service.NewStorageService(orderRepo, userRepo, reportRepo)
	authService := service.NewAuthService(authRepo)

	apiHandler := api.NewAPIHandler(storageService)
	authHandler := api.NewAuthHandler(authService, logger)

	router := router.SetupRouter(apiHandler, authHandler, logger)

	if err := router.Run(cfg.HTTPPort); err != nil {
		log.Fatalf("Сервер не запустился: %v", err)
	}

}
