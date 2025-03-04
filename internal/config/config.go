package config

import (
	"github.com/joho/godotenv"
	"log"
	"os"
)

type Config struct {
	OrderStoragePath string
}

func Load() *Config {
	if err := godotenv.Load(); err != nil {
		log.Fatalf("Ошибка загрузки .env файла: %v", err)
	}

	return &Config{
		OrderStoragePath: getEnv("ORDER_STORAGE_PATH", "./data/orders.json"),
	}
}

func getEnv(key, defaultValue string) string {
	value, exists := os.LookupEnv(key)
	if !exists {
		return defaultValue
	}
	return value
}
