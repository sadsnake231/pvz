package audit

import (
	"context"
	"strings"

	"gitlab.ozon.dev/sadsnake2311/homework/internal/domain"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/service"
	"go.uber.org/zap"
)

type FilterFunc func(domain.Event) bool

type Pipeline struct {
	pools   []*WorkerPool
	service service.AuditService
	logger  *zap.SugaredLogger
}

func NewPipeline(service service.AuditService, logger *zap.SugaredLogger, pools ...*WorkerPool) *Pipeline {
	return &Pipeline{
		pools:   pools,
		service: service,
		logger:  logger,
	}
}
func (p *Pipeline) StartWorkers(ctx context.Context, filterFunc FilterFunc) error {
	p.pools[0].StartWorkers(ctx, p.saveToDB(ctx))

	p.pools[1].StartWorkers(ctx, p.printToStdout(filterFunc))

	return nil
}

func (p *Pipeline) saveToDB(ctx context.Context) func(domain.Event) error {
	return func(e domain.Event) error {
		err := p.service.SaveLog(ctx, e)
		return err
	}
}

func (p *Pipeline) printToStdout(filterFunc FilterFunc) func(domain.Event) error {
	return func(e domain.Event) error {
		if filterFunc(e) {
			p.logger.Infow("[AUDIT]",
				"event_type", e.Type,
				"data", e.Data,
			)
		}
		return nil
	}
}

func (p *Pipeline) SendEvent(eventType domain.EventType, data any) {
	event := domain.NewEvent(eventType, data)

	for _, pool := range p.pools {
		switch eventType {
		case domain.EventAPIRequest, domain.EventAPIResponse:
			go func(pool *WorkerPool) { pool.ApiChan <- event }(pool)
		case domain.EventStatusChange:
			go func(pool *WorkerPool) { pool.StatusChan <- event }(pool)
		}
	}
}

func NewFilterFunc(filterKeyword string) FilterFunc {
	return func(e domain.Event) bool {
		if filterKeyword == "" {
			return true
		}

		if data, ok := e.Data.(map[string]any); ok {
			for key, value := range data {
				if str, ok := value.(string); ok {
					if strings.Contains(str, filterKeyword) {
						return true
					}
				}
				if strings.Contains(key, filterKeyword) {
					return true
				}
			}
		}

		return false
	}
}
