package audit

import (
	"context"

	"gitlab.ozon.dev/sadsnake2311/homework/internal/domain"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/service"
	"go.uber.org/zap"
)

type FilterFunc func(domain.Event) bool

type Pipeline struct {
	DbPool     *WorkerPool
	StdoutPool *WorkerPool
	filterFunc FilterFunc
	service    service.AuditService
	logger     *zap.SugaredLogger
}

func NewPipeline(filterFunc FilterFunc, service service.AuditService, logger *zap.SugaredLogger) *Pipeline {
	return &Pipeline{
		DbPool:     NewWorkerPool(logger),
		StdoutPool: NewWorkerPool(logger),
		filterFunc: filterFunc,
		service:    service,
		logger:     logger,
	}
}

func (p *Pipeline) StartWorkers(ctx context.Context) {
	p.DbPool.StartWorkers(ctx,
		func(e domain.Event) error { return p.saveToDB(e) }, // Для статусов
		func(e domain.Event) error { return p.saveToDB(e) }, // Для API
	)

	p.StdoutPool.StartWorkers(ctx,
		func(e domain.Event) error { return p.printToStdout(e) }, // Для статусов
		func(e domain.Event) error { return p.printToStdout(e) }, // Для API
	)
}

func (p *Pipeline) saveToDB(e domain.Event) error {
	err := p.service.SaveLog(context.Background(), e)
	//fmt.Printf("[DB] Saving event: %+v\n", e)
	return err
}

func (p *Pipeline) printToStdout(e domain.Event) error {
	if p.filterFunc(e) {
		p.logger.Infow("[AUDIT]",
			"event_type", e.Type,
			"data", e.Data,
		)
	}
	return nil
}

func NewFilterFunc(filterKeyword string) FilterFunc {
	return func(e domain.Event) bool {
		if filterKeyword == "" {
			return true
		}

		if data, ok := e.Data.(map[string]any); ok {
			for _, value := range data {
				if str, ok := value.(string); ok && str == filterKeyword {
					return true
				}
			}
		}

		return false
	}
}
