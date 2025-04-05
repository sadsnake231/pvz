-- +goose Up
-- +goose StatementBegin
ALTER TABLE audit_logs RENAME TO audit_logs_old;

CREATE TABLE audit_tasks(
    id SERIAL PRIMARY KEY,
    audit_log JSONB NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('CREATED', 'PROCESSING', 'FAILED', 'FINISHED', 'NO_ATTEMPTS_LEFT')),
    attempts_left INT NOT NULL,
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    finished_at TIMESTAMP,
    next_retry TIMESTAMP
);

INSERT INTO audit_tasks (
    audit_log,
    status,
    created_at,
    updated_at
)
SELECT
    jsonb_build_object(
        'type', event_type,
        'data', event_data
    ),
    'CREATED',
    created_at,
    created_at
FROM audit_logs_old;

DROP TABLE audit_logs_old

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
CREATE TABLE audit_logs (
    id SERIAL PRIMARY KEY,
    event_type TEXT NOT NULL,    
    event_data JSONB NOT NULL,
    created_at TIMESTAMPTZ
);

INSERT INTO audit_logs (event_type, event_data, created_at)
SELECT
    (audit_log->>'type')::TEXT,
    (audit_log->'data')::JSONB,
    created_at
FROM audit_tasks;

DROP TABLE audit_tasks
-- +goose StatementEnd
