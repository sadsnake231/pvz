package auditrepo

import (
	"context"

	"github.com/jackc/pgx/v5"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/domain"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/storage"
	"go.uber.org/zap"
)

type AuditRepository interface {
	SaveLog(ctx context.Context, auditTask domain.AuditTask) error
	BeginTx(ctx context.Context) (pgx.Tx, error)
	FetchPendingTasksTx(ctx context.Context, tx pgx.Tx, limit int) ([]domain.AuditTask, error)
	UpdateTask(ctx context.Context, task domain.AuditTask) error
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

func (r *auditRepository) BeginTx(ctx context.Context) (pgx.Tx, error) {
	return r.storage.BeginTx(ctx)
}

func (r *auditRepository) FetchPendingTasksTx(ctx context.Context, tx pgx.Tx, limit int) ([]domain.AuditTask, error) {
	return r.storage.FetchPendingTasksTx(ctx, tx, limit)
}

func (r *auditRepository) UpdateTask(ctx context.Context, task domain.AuditTask) error {
	return r.storage.UpdateTask(ctx, task)
}
