-- +goose Up
-- +goose StatementBegin
ALTER TABLE ledgers
ADD CONSTRAINT ledger_name_unique UNIQUE (name);

CREATE INDEX idx_ledgers_name ON ledgers(name);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE ledgers
DROP CONSTRAINT ledger_name_unique;
-- +goose StatementEnd
