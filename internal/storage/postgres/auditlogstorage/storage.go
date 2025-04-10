package auditlogstorage

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/domain"
)

type AuditLogStorage struct {
	db *pgxpool.Pool
}

func NewAuditStorage(db *pgxpool.Pool) *AuditLogStorage {
	return &AuditLogStorage{db: db}
}

func (s *AuditLogStorage) SaveLog(ctx context.Context, auditTask domain.AuditTask) error {
	query := `
        INSERT INTO audit_tasks 
        (audit_log, status, attempt_number, created_at, updated_at) 
        VALUES ($1, $2, $3, $4, $5)
    `
	_, err := s.db.Exec(ctx, query,
		auditTask.AuditLog,
		auditTask.Status,
		auditTask.AttemptNumber,
		auditTask.CreatedAt,
		auditTask.UpdatedAt,
	)
	return err
}

func (s *AuditLogStorage) BeginTx(ctx context.Context) (pgx.Tx, error) {
	return s.db.Begin(ctx)
}

func (s *AuditLogStorage) FetchPendingTasksTx(ctx context.Context, tx pgx.Tx, limit int) ([]domain.AuditTask, error) {
	query := `
		SELECT id, audit_log, status, attempt_number,
		       created_at, updated_at,
		       COALESCE(finished_at, '0001-01-01'::timestamp),
		       COALESCE(next_retry, '0001-01-01'::timestamp)
		FROM audit_tasks
		WHERE status IN ('CREATED', 'FAILED')
		  AND (next_retry IS NULL OR next_retry < NOW())
		  AND attempt_number < 3
		ORDER BY created_at
		LIMIT $1
		FOR UPDATE SKIP LOCKED
	`

	rows, err := tx.Query(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []domain.AuditTask
	for rows.Next() {
		var task domain.AuditTask
		if err := rows.Scan(
			&task.ID,
			&task.AuditLog,
			&task.Status,
			&task.AttemptNumber,
			&task.CreatedAt,
			&task.UpdatedAt,
			&task.FinishedAt,
			&task.NextRetry,
		); err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}

	return tasks, nil
}

func (s *AuditLogStorage) UpdateTask(ctx context.Context, task domain.AuditTask) error {
	query := `
		UPDATE audit_tasks
		SET status = $1,
			attempt_number = $2,
			updated_at = $3,
			next_retry = $4
		WHERE id = $5
	`

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx,
		query,
		task.Status,
		task.AttemptNumber,
		task.UpdatedAt,
		task.NextRetry,
		task.ID,
	); err != nil {
		return err
	}

	return tx.Commit(ctx)
}
