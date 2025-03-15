-- +goose Up
-- +goose StatementBegin

ALTER TABLE orders ADD COLUMN order_id VARCHAR(36);

UPDATE orders SET order_id = id;

ALTER TABLE orders ALTER COLUMN order_id SET NOT NULL;

ALTER TABLE orders DROP CONSTRAINT orders_pkey;

ALTER TABLE orders ADD COLUMN new_id SERIAL;

ALTER TABLE orders ADD PRIMARY KEY (new_id);

ALTER TABLE orders DROP COLUMN id;

ALTER TABLE orders RENAME COLUMN new_id TO id;

CREATE UNIQUE INDEX idx_orders_order_id ON orders(order_id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_orders_order_id;

ALTER TABLE orders DROP COLUMN order_id;

ALTER TABLE orders DROP CONSTRAINT orders_pkey;

ALTER TABLE orders ADD COLUMN id VARCHAR(36);

ALTER TABLE orders ADD PRIMARY KEY (id);

ALTER TABLE orders DROP COLUMN id;

-- +goose StatementEnd