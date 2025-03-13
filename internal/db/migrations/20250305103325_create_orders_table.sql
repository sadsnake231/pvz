-- +goose Up
-- +goose StatementBegin
CREATE TABLE packaging_types(
    id VARCHAR(36) PRIMARY KEY,
    packaging_price NUMERIC(10, 2) NOT NULL
);

CREATE TABLE orders(
    id VARCHAR(36) PRIMARY KEY,
    recipient_id VARCHAR(36) NOT NULL,
    expiry TIMESTAMP NOT NULL,
    stored_at TIMESTAMP,
    issued_at TIMESTAMP,
    refunded_at TIMESTAMP,
    base_price NUMERIC(10, 2) NOT NULL,
    weight NUMERIC(10, 2) NOT NULL,
    packaging VARCHAR(50) NOT NULL REFERENCES packaging_types(id)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS orders CASCADE;
DROP TABLE IF EXISTS packaging_types CASCADE;
-- +goose StatementEnd
