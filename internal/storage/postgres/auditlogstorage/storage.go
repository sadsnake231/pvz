package auditlogstorage

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5/pgxpool"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/domain"
)

type AuditLogStorage struct {
	db *pgxpool.Pool
}

func NewAuditStorage(db *pgxpool.Pool) *AuditLogStorage {
	return &AuditLogStorage{db: db}
}

func (s *AuditLogStorage) SaveLog(ctx context.Context, event domain.Event) error {
	query := `
		INSERT INTO audit_logs 
		(event_type, event_data, created_at) 
		VALUES ($1, $2, $3)
	`
	_, err := s.db.Exec(ctx, query, event.Type, event.Data, event.Time)
	return err
}

func (s *AuditLogStorage) GetLogs(ctx context.Context, limit int, cursor *int) ([]domain.Event, int, error) {
	query := `
		SELECT *
		FROM audit_logs
		WHERE ($1::INT IS NULL OR id < $1)
		ORDER BY id DESC
		LIMIT $2
	`
	var nextCursor int

	rows, err := s.db.Query(ctx, query, cursor, limit)
	if err != nil {
		return nil, nextCursor, err
	}
	defer rows.Close()

	var events []domain.Event
	for rows.Next() {
		var event domain.Event
		var eventData []byte
		if err := rows.Scan(
			&nextCursor,
			&event.Type,
			&eventData,
			&event.Time,
		); err != nil {
			return nil, nextCursor, err
		}

		if err := json.Unmarshal(eventData, &event.Data); err != nil {
			return nil, 0, err
		}

		events = append(events, event)
	}
	return events, nextCursor, nil
}
