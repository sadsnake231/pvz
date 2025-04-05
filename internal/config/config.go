package config

import (
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL   string
	HTTPPort      string
	AuditFilter   string
	CacheURL      string
	CachePassword string
	KafkaBrokers  []string
}

func Load() *Config {
	if err := godotenv.Load(); err != nil {
		log.Fatalf("Ошибка загрузки .env файла: %v", err)
	}

	return &Config{
		DatabaseURL:   getEnv("DATABASE_URL", ""),
		HTTPPort:      getEnv("HTTP_PORT", ":9000"),
		AuditFilter:   getEnv("AUDIT_FILTER", ""),
		CacheURL:      getEnv("CACHE_URL", ""),
		CachePassword: getEnv("CACHE_PASSWORD", ""),
		KafkaBrokers:  strings.Split(getEnv("KAFKA_BROKERS", ""), ","),
	}
}

func getEnv(key, defaultValue string) string {
	value, exists := os.LookupEnv(key)
	if !exists {
		return defaultValue
	}
	return value
}
