package auditrepo

import (
	"context"

	"gitlab.ozon.dev/sadsnake2311/homework/internal/domain"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/kafka"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/storage"
	"go.uber.org/zap"
)

type AuditRepository interface {
	SaveLog(ctx context.Context, auditTask domain.AuditTask) error
	FetchPendingTasks(ctx context.Context, limit int) ([]domain.AuditTask, error)
	UpdateTask(ctx context.Context, task domain.AuditTask) error
	ProcessTaskWithKafka(ctx context.Context, task domain.AuditTask, kafkaProducer *kafka.Producer) error
}

type auditRepository struct {
	storage storage.AuditLogStorage
	logger  *zap.SugaredLogger
}

func NewAuditRepository(storage storage.AuditLogStorage, logger *zap.SugaredLogger) AuditRepository {
	return &auditRepository{
		storage: storage,
		logger:  logger,
	}
}

func (r *auditRepository) SaveLog(ctx context.Context, auditTask domain.AuditTask) error {
	return r.storage.SaveLog(ctx, auditTask)
}

func (r *auditRepository) FetchPendingTasks(ctx context.Context, limit int) ([]domain.AuditTask, error) {
	return r.storage.FetchPendingTasks(ctx, limit)
}

func (r *auditRepository) UpdateTask(ctx context.Context, task domain.AuditTask) error {
	return r.storage.UpdateTask(ctx, task)
}

func (r *auditRepository) ProcessTaskWithKafka(ctx context.Context, task domain.AuditTask, kafkaProducer *kafka.Producer) error {
	return r.storage.ProcessTaskWithKafka(ctx, task, kafkaProducer)
}
