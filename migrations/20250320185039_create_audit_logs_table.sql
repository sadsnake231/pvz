-- +goose Up
-- +goose StatementBegin
CREATE TABLE audit_logs (
    id SERIAL PRIMARY KEY,
    event_type VARCHAR(50) NOT NULL,    
    event_data JSONB NOT NULL,
    created_at TIMESTAMP
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS audit_logs CASCADE;
-- +goose StatementEnd
