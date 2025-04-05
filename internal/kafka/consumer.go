package kafka

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/twmb/franz-go/pkg/kgo"
	"go.uber.org/zap"
)

type Consumer struct {
	client   *kgo.Client
	producer *Producer
	groupID  string
	logger   *zap.SugaredLogger
}

func NewConsumer(brokers []string, groupID string, producer *Producer, logger *zap.SugaredLogger) (*Consumer, error) {
	opts := []kgo.Opt{
		kgo.SeedBrokers(brokers...),
		kgo.ConsumerGroup(groupID),
		kgo.ConsumeTopics("audit_logs"),
		kgo.FetchIsolationLevel(kgo.ReadCommitted()),
		kgo.DisableAutoCommit(),
		kgo.ConsumeResetOffset(kgo.NewOffset().AtEnd()),

		kgo.SessionTimeout(30 * time.Second),
		kgo.HeartbeatInterval(5 * time.Second),

		kgo.OnPartitionsRevoked(func(ctx context.Context, client *kgo.Client, revoked map[string][]int32) {
			if err := client.CommitUncommittedOffsets(ctx); err != nil {
				logger.Errorw("Failed to commit offsets during rebalance", "error", err)
			}
		}),
		kgo.OnPartitionsAssigned(func(ctx context.Context, client *kgo.Client, assigned map[string][]int32) {
			logger.Infow("Partitions assigned", "partitions", assigned)
		}),
	}

	client, err := kgo.NewClient(opts...)
	if err != nil {
		return nil, fmt.Errorf("не смог запустить клиент Кафки: %w", err)
	}

	return &Consumer{
		client:   client,
		producer: producer,
		groupID:  groupID,
		logger:   logger,
	}, nil
}

func (c *Consumer) Run(ctx context.Context) error {
	defer c.client.Close()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-sigCh:
			return nil
		default:
			fetches := c.client.PollFetches(ctx)
			if fetches.IsClientClosed() {
				return fmt.Errorf("клиент закрыт")
			}

			if errs := fetches.Errors(); len(errs) > 0 {
				for _, err := range errs {
					c.logger.Errorw("Fetch error",
						"topic", err.Topic,
						"partition", err.Partition,
						"error", err.Err,
					)
				}
				continue
			}

			fetches.EachRecord(func(record *kgo.Record) {
				if err := c.processRecord(record); err != nil {
					c.logger.Errorw("Ошибка обработки записи",
						"topic", record.Topic,
						"partition", record.Partition,
						"offset", record.Offset,
						"error", err,
					)
					return
				}

				c.client.MarkCommitRecords(record)
			})

			if err := c.client.CommitUncommittedOffsets(ctx); err != nil {
				c.logger.Errorw("Failed to commit offsets", "error", err)
			}
		}
	}
}

func (c *Consumer) processRecord(record *kgo.Record) error {
	defer func() {
		if r := recover(); r != nil {
			c.logger.Errorw("Паника при обработке записи",
				"offset", record.Offset,
				"recover", r,
			)
		}
	}()

	c.logger.Infow("[KAFKA AUDIT]",
		"topic", record.Topic,
		"partition", record.Partition,
		"offset", record.Offset,
		"value", string(record.Value),
	)

	return nil
}
