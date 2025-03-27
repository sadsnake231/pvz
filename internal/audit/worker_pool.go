package audit

import (
	"context"

	"gitlab.ozon.dev/sadsnake2311/homework/internal/domain"
	"go.uber.org/zap"
)

type WorkerPool struct {
	StatusChan chan domain.Event
	ApiChan    chan domain.Event
	logger     *zap.SugaredLogger
}

func NewWorkerPool(logger *zap.SugaredLogger) *WorkerPool {
	return &WorkerPool{
		StatusChan: make(chan domain.Event, 1000),
		ApiChan:    make(chan domain.Event, 1000),
		logger:     logger,
	}
}

func (p *WorkerPool) StartWorkers(ctx context.Context, handler func(domain.Event) error) {
	go NewWorker(p.ApiChan, handler, "api_worker", p.logger).Run(ctx)

	go NewWorker(p.StatusChan, handler, "status_worker", p.logger).Run(ctx)
}
