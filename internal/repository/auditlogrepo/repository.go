package auditrepo

import (
	"context"

	"gitlab.ozon.dev/sadsnake2311/homework/internal/domain"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/storage"
	"go.uber.org/zap"
)

type AuditRepository interface {
	SaveLog(ctx context.Context, event domain.Event) error
	GetLogs(ctx context.Context, limit int, cursor *int) ([]domain.Event, int, error)
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

func (r *auditRepository) SaveLog(ctx context.Context, event domain.Event) error {
	return r.storage.SaveLog(ctx, event)
}

func (r *auditRepository) GetLogs(ctx context.Context, limit int, cursor *int) ([]domain.Event, int, error) {
	logs, nextCursor, err := r.storage.GetLogs(ctx, limit, cursor)
	if err != nil {
		r.logger.Error("ошибка вывода аудит логов", zap.Error(err))
		return logs, nextCursor, domain.ErrDatabase
	}

	return logs, nextCursor, nil
}
