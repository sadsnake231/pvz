package service

import (
	"context"
	"encoding/json"

	"gitlab.ozon.dev/sadsnake2311/homework/internal/domain"
	repository "gitlab.ozon.dev/sadsnake2311/homework/internal/repository/auditlogrepo"
)

type AuditService interface {
	SaveLog(ctx context.Context, event domain.Event) error
	GetLogs(ctx context.Context, limit int, cursor *int) ([]domain.Event, int, error)
}

type auditService struct {
	repo repository.AuditRepository
}

func NewAuditService(repo repository.AuditRepository) AuditService {
	return &auditService{repo: repo}
}

func (s *auditService) SaveLog(ctx context.Context, event domain.Event) error {
	dataJSON, err := json.Marshal(event.Data)
	if err != nil {
		return err
	}

	event.Data = dataJSON

	return s.repo.SaveLog(ctx, event)
}

func (s *auditService) GetLogs(ctx context.Context, limit int, cursor *int) ([]domain.Event, int, error) {
	events, nextCursor, err := s.repo.GetLogs(ctx, limit, cursor)
	if err != nil {
		return nil, nextCursor, err
	}

	result := make([]domain.Event, 0, len(events))
	for _, event := range events {

		result = append(result, domain.Event{
			Type: event.Type,
			Data: event.Data,
			Time: event.Time,
		})
	}

	return result, nextCursor, nil
}
