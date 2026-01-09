-- +goose Up
-- +goose StatementBegin
ALTER TABLE ledgers
ADD COLUMN currency VARCHAR(3) NOT NULL
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE ledgers
DROP COLUMN currency
-- +goose StatementEnd
