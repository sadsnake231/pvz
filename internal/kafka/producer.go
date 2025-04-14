package kafka

import (
	"context"
	"strconv"
	"time"

	"github.com/twmb/franz-go/pkg/kgo"
	"go.uber.org/zap"
)

type Producer struct {
	client *kgo.Client
	logger *zap.SugaredLogger
}

func NewProducer(brokers []string, logger *zap.SugaredLogger) (*Producer, error) {

	opts := []kgo.Opt{
		kgo.SeedBrokers(brokers...),
		kgo.TransactionalID("audit-producer-v1"),
		kgo.AllowAutoTopicCreation(),
		kgo.ProduceRequestTimeout(3 * time.Second),
		kgo.ProducerBatchMaxBytes(10 << 20),
		kgo.ProducerLinger(100),
	}

	client, err := kgo.NewClient(opts...)
	if err != nil {
		return nil, err
	}

	return &Producer{
		client: client,
		logger: logger,
	}, nil
}

func (p *Producer) SendTransactional(
	ctx context.Context,
	taskID int,
	payload []byte,
) error {

	if err := p.client.BeginTransaction(); err != nil {
		p.logger.Errorw("producer failed to start a transaction", "error", err)
		return err
	}

	record := &kgo.Record{
		Topic: "audit_logs",
		Key:   []byte(strconv.Itoa(taskID)),
		Value: payload,
	}

	result := p.client.ProduceSync(ctx, record)
	if err := result.FirstErr(); err != nil {
		_ = p.client.EndTransaction(ctx, kgo.TryAbort)
		p.logger.Errorw("producer failed to send a transaction", "error", err)
		return err
	}

	if err := p.client.EndTransaction(ctx, kgo.TryCommit); err != nil {
		p.logger.Errorw("producer failed to commit a transaction", "error", err)
		return err
	}

	return nil
}
