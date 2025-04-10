package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"gitlab.ozon.dev/sadsnake2311/homework/internal/config"
	"go.uber.org/zap"
)

func main() {
	baseLogger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("Ошибка старта логгера: %v", err)
	}
	logger := baseLogger.Sugar()
	defer logger.Sync()

	cfg := config.Load()

	kafkaConsumer, err := NewConsumer(cfg.KafkaBrokers, cfg.KafkaConsumerGroup, cfg.KafkaTopic, logger)
	if err != nil {
		logger.Fatalw("Kafka Consumer не запустился", "error", err)
	}
	defer kafkaConsumer.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	kafkaConsumer.Start(ctx)

	select {
	case <-sigChan:
		cancel()
	case <-ctx.Done():
	}

	kafkaConsumer.Wait()
	logger.Info("Consumer завершил работу")
}
