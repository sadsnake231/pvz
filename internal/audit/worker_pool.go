package audit

import (
	"context"

	"gitlab.ozon.dev/sadsnake2311/homework/internal/domain"
)

type WorkerPool struct {
	StatusChan chan domain.Event
	ApiChan    chan domain.Event
}

func NewWorkerPool() *WorkerPool {
	return &WorkerPool{
		StatusChan: make(chan domain.Event, 1000),
		ApiChan:    make(chan domain.Event, 1000),
	}
}

func (p *WorkerPool) StartWorkers(ctx context.Context, processStatusFunc, processAPIFunc func(domain.Event) error) {
	// StatusWorker
	go NewWorker(p.StatusChan, processStatusFunc, "status_worker").Run(ctx)

	// APIWorker
	go NewWorker(p.ApiChan, processAPIFunc, "api_worker").Run(ctx)
}
