package audit

import (
	"context"
	"fmt"
	"time"

	"gitlab.ozon.dev/sadsnake2311/homework/internal/domain"
)

type Worker struct {
	inputChan   <-chan domain.Event
	processFunc func(domain.Event) error
	workerType  string
}

func NewWorker(input <-chan domain.Event, f func(domain.Event) error, workerType string) *Worker {
	return &Worker{
		inputChan:   input,
		processFunc: f,
		workerType:  workerType,
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
			fmt.Printf("[%s] Ошибка обработки: %v\n", w.workerType, err)
		}
	}
}
