package audit

import (
	"context"
	"time"

	"gitlab.ozon.dev/sadsnake2311/homework/internal/domain"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/kafka"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/service"
	"go.uber.org/zap"
)

type OutboxWorker struct {
	service       service.AuditService
	kafkaProducer *kafka.Producer
	logger        *zap.SugaredLogger
	pollInterval  time.Duration
	batchSize     int
	retryDelay    time.Duration
	maxAttempts   int
}

func NewOutboxWorker(
	service service.AuditService,
	producer *kafka.Producer,
	logger *zap.SugaredLogger,
) *OutboxWorker {
	return &OutboxWorker{
		service:       service,
		kafkaProducer: producer,
		logger:        logger,
		pollInterval:  500 * time.Millisecond,
		batchSize:     100,
		retryDelay:    2 * time.Second,
		maxAttempts:   3,
	}
}

func (w *OutboxWorker) Run(ctx context.Context) {
	w.logger.Info("Outbox worker запущен")
	defer w.logger.Info("Outbox worker остановлен")

	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.processBatch(ctx)
		}
	}
}

func (w *OutboxWorker) processBatch(ctx context.Context) {
	tx, err := w.service.BeginTx(ctx)
	if err != nil {
		w.logger.Errorw("Не смог начать транзакцию", "error", err)
		return
	}
	defer tx.Rollback(ctx)

	tasks, err := w.service.FetchPendingTasksTx(ctx, tx, w.batchSize)
	if err != nil {
		w.logger.Errorw("Не смог получить таски", "error", err)
		return
	}

	if err := tx.Commit(ctx); err != nil {
		w.logger.Errorw("Не смог закоммитить транзакцию", "error", err)
		return
	}

	if len(tasks) == 0 {
		return
	}

	for _, task := range tasks {
		select {
		case <-ctx.Done():
			return
		default:
			if err := w.processTask(ctx, task); err != nil {
				w.logger.Errorw("Не смог обработать таск",
					"task_id", task.ID,
					"error", err)
			}
		}
	}
}

func (w *OutboxWorker) processTask(ctx context.Context, task domain.AuditTask) error {
	now := time.Now().UTC()

	task.Status = domain.StatusProcessing
	task.UpdatedAt = now
	if err := w.service.UpdateTask(ctx, task); err != nil {
		return err
	}

	kafkaCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	err := w.kafkaProducer.SendTransactional(kafkaCtx, task.ID, task.AuditLog)
	if err != nil {
		task.AttemptNumber++
		task.Status = domain.StatusFailed
		task.NextRetry = now.Add(w.retryDelay)
		task.UpdatedAt = now

		if task.AttemptNumber >= w.maxAttempts {
			task.Status = domain.StatusNoAttemptsLeft
		}

		return w.service.UpdateTask(ctx, task)
	}

	task.Status = domain.StatusFinished
	task.UpdatedAt = now
	task.FinishedAt = now
	return w.service.UpdateTask(ctx, task)
}
