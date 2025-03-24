package audit

import (
	"context"
	"time"

	"gitlab.ozon.dev/sadsnake2311/homework/internal/domain"
	"go.uber.org/zap"
)

type Worker struct {
	inputChan   <-chan domain.Event
	processFunc func(domain.Event) error
	workerType  string
	logger      *zap.SugaredLogger
}

func NewWorker(input <-chan domain.Event, f func(domain.Event) error, workerType string, logger *zap.SugaredLogger) *Worker {
	return &Worker{
		inputChan:   input,
		processFunc: f,
		workerType:  workerType,
		logger:      logger,
	}
}

func (w *Worker) Run(ctx context.Context) {
	batch := make([]domain.Event, 0, 5)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case event := <-w.inputChan:
			batch = append(batch, event)
			if len(batch) >= 5 {
				w.processBatch(batch)
				batch = nil
			}

		case <-ticker.C:
			if len(batch) > 0 {
				w.processBatch(batch)
				batch = nil
			}

		case <-ctx.Done():
			if len(batch) > 0 {
				w.processBatch(batch)
			}
			return
		}
	}
}

func (w *Worker) processBatch(events []domain.Event) {
	for _, event := range events {
		if err := w.processFunc(event); err != nil {
			w.logger.Errorw("Ошибка обработки события",
				"worker_type", w.workerType,
				"error", err,
			)
		}
	}
}
