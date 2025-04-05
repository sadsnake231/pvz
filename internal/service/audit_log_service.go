package service

import (
	"context"
	"encoding/json"
	"time"

	"gitlab.ozon.dev/sadsnake2311/homework/internal/domain"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/kafka"
	repository "gitlab.ozon.dev/sadsnake2311/homework/internal/repository/auditlogrepo"
)

type AuditService interface {
	SaveLog(ctx context.Context, event domain.Event) error
	FetchPendingTasks(ctx context.Context, limit int) ([]domain.AuditTask, error)
	UpdateTask(ctx context.Context, task domain.AuditTask) error
	ProcessTaskWithKafka(ctx context.Context, task domain.AuditTask, kafkaProducer *kafka.Producer) error
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
		AuditLog:     auditLogJSON,
		Status:       "CREATED",
		AttemptsLeft: 3,
		CreatedAt:    event.Time,
		UpdatedAt:    event.Time,
	}

	return s.repo.SaveLog(ctx, auditTask)
}

func (s *auditService) FetchPendingTasks(ctx context.Context, limit int) ([]domain.AuditTask, error) {
	return s.repo.FetchPendingTasks(ctx, limit)
}

func (s *auditService) UpdateTask(ctx context.Context, task domain.AuditTask) error {
	task.UpdatedAt = time.Now()

	return s.repo.UpdateTask(ctx, task)
}

func (s *auditService) ProcessTaskWithKafka(ctx context.Context, task domain.AuditTask, kafkaProducer *kafka.Producer) error {
	return s.repo.ProcessTaskWithKafka(ctx, task, kafkaProducer)
}
