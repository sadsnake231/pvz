package config

import (
	"log"
	"os"
	"github.com/joho/godotenv"
)

type Config struct {
	OrderStoragePath  string
}

func Load() *Config {
	// Загружаем .env файл
	if err := godotenv.Load(); err != nil {
		log.Fatalf("Ошибка загрузки .env файла: %v", err)
	}

	return &Config{
		OrderStoragePath:  getEnv("ORDER_STORAGE_PATH", "./data/orders.json"),
	}
}

// getEnv возвращает значение переменной окружения или значение по умолчанию
func getEnv(key, defaultValue string) string {
	value, exists := os.LookupEnv(key)
	if !exists {
		return defaultValue
	}
	return value
}
