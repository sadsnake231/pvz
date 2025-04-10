package service

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/domain"
	repository "gitlab.ozon.dev/sadsnake2311/homework/internal/repository/auditlogrepo"
)

type AuditService interface {
	SaveLog(ctx context.Context, event domain.Event) error
	BeginTx(ctx context.Context) (pgx.Tx, error)
	FetchPendingTasksTx(ctx context.Context, tx pgx.Tx, limit int) ([]domain.AuditTask, error)
	UpdateTask(ctx context.Context, task domain.AuditTask) error
}

type auditService struct {
	repo repository.AuditRepository
}

func NewAuditService(repo repository.AuditRepository) AuditService {
	return &auditService{repo: repo}
}

func (s *auditService) SaveLog(ctx context.Context, event domain.Event) error {
	auditLogJSON, err := json.Marshal(event)
	if err != nil {
		return err
	}

	auditTask := domain.AuditTask{
		AuditLog:      auditLogJSON,
		Status:        "CREATED",
		AttemptNumber: 0,
		CreatedAt:     event.Time,
		UpdatedAt:     event.Time,
	}

	return s.repo.SaveLog(ctx, auditTask)
}

func (s *auditService) BeginTx(ctx context.Context) (pgx.Tx, error) {
	return s.repo.BeginTx(ctx)
}

func (s *auditService) FetchPendingTasksTx(ctx context.Context, tx pgx.Tx, limit int) ([]domain.AuditTask, error) {
	return s.repo.FetchPendingTasksTx(ctx, tx, limit)
}

func (s *auditService) UpdateTask(ctx context.Context, task domain.AuditTask) error {
	task.UpdatedAt = time.Now()

	return s.repo.UpdateTask(ctx, task)
}
