package auditlogstorage

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/domain"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/kafka"
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
        (audit_log, status, attempts_left, created_at, updated_at) 
        VALUES ($1, $2, $3, $4, $5)
    `
	_, err := s.db.Exec(ctx, query,
		auditTask.AuditLog,
		auditTask.Status,
		auditTask.AttemptsLeft,
		auditTask.CreatedAt,
		auditTask.UpdatedAt,
	)
	return err
}

func (s *AuditLogStorage) FetchPendingTasks(ctx context.Context, limit int) ([]domain.AuditTask, error) {
	query := `
		SELECT id, audit_log, status, attempts_left,
				created_at, updated_at,
				COALESCE(finished_at, '0001-01-01'::timestamp) as finished_at,
            	COALESCE(next_retry, '0001-01-01'::timestamp) as next_retry
		FROM audit_tasks
		WHERE status IN ('CREATED', 'FAILED')
		AND (next_retry IS NULL OR next_retry < NOW())
		AND attempts_left > 0
		ORDER BY created_at
		LIMIT $1
		FOR UPDATE SKIP LOCKED
	`

	rows, err := s.db.Query(ctx, query, limit)
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
			&task.AttemptsLeft,
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
			attempts_left = $2,
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
		task.AttemptsLeft,
		task.UpdatedAt,
		task.NextRetry,
		task.ID,
	); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (s *AuditLogStorage) ProcessTaskWithKafka(
	ctx context.Context,
	task domain.AuditTask,
	kafkaProducer *kafka.Producer,
) error {
	task.Status = domain.StatusProcessing
	task.UpdatedAt = time.Now().UTC()

	err := s.UpdateTask(ctx, task)
	if err != nil {
		return err
	}

	kafkaCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	err = kafkaProducer.SendTransactional(kafkaCtx, task.ID, task.AuditLog)
	if err != nil {
		task.AttemptsLeft--
		task.Status = domain.StatusFailed
		task.NextRetry = time.Now().UTC().Add(2 * time.Second)

		if task.AttemptsLeft <= 0 {
			task.Status = domain.StatusNoAttemptsLeft
		}
		return s.UpdateTask(ctx, task)
	}

	task.Status = domain.StatusFinished
	task.UpdatedAt = time.Now().UTC()
	task.FinishedAt = time.Now().UTC()
	return s.UpdateTask(ctx, task)
}
