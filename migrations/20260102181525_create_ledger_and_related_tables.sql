-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS ledgers (
    id UUID PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    created_by UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_ledgers_created_by ON ledgers(created_by);

CREATE TABLE IF NOT EXISTS ledger_users (
    ledger_id UUID NOT NULL REFERENCES ledgers(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    PRIMARY KEY (ledger_id, user_id)
);

CREATE INDEX idx_ledger_users_user_id ON ledger_users(user_id);

CREATE TABLE IF NOT EXISTS ledger_expenses (
    id UUID PRIMARY KEY,
    ledger_id UUID NOT NULL REFERENCES ledgers(id) ON DELETE CASCADE,
    description VARCHAR(500) NOT NULL,
    amount BIGINT NOT NULL,
    paid_by UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    split_type VARCHAR(50) NOT NULL,
    category VARCHAR(100) NOT NULL,
    created_at TIMESTAMP NOT NULL
);

CREATE INDEX idx_ledger_expenses_ledger_id ON ledger_expenses(ledger_id);
CREATE INDEX idx_ledger_expenses_paid_by ON ledger_expenses(paid_by);
CREATE INDEX idx_ledger_expenses_created_at ON ledger_expenses(created_at DESC);

CREATE TABLE IF NOT EXISTS ledger_expense_splits (
    expense_id UUID NOT NULL REFERENCES ledger_expenses(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    amount BIGINT NOT NULL,
    PRIMARY KEY (expense_id, user_id)
);

CREATE INDEX idx_ledger_expense_splits_user_id ON ledger_expense_splits(user_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS ledgers;
DROP TABLE IF EXISTS ledger_users;
DROP TABLE IF EXISTS ledger_expenses;
DROP TABLE IF EXISTS ledger_expense_splits;
-- +goose StatementEnd
