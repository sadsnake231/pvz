-- +goose Up
-- +goose StatementBegin
CREATE TABLE audit_tasks(
    id SERIAL PRIMARY KEY,
    audit_log JSONB NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('CREATED', 'PROCESSING', 'FAILED', 'FINISHED', 'NO_ATTEMPTS_LEFT')),
    attempt_number INT NOT NULL,
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    finished_at TIMESTAMP,
    next_retry TIMESTAMP
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS audit_tasks;
-- +goose StatementEnd
