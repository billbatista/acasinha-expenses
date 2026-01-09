package ledger

import (
	"context"
	"database/sql"
)

type repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *repository {
	return &repository{db: db}
}

func (r *repository) CreateNew(ctx context.Context, ledger Ledger) (string, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	var lastId string
	if err != nil {
		return lastId, err
	}
	defer tx.Rollback()

	insertLedger := `INSERT INTO ledgers (id, name, currency, created_by, created_at) VALUES ($1, $2, $3, $4, $5) RETURNING id`
	err = tx.QueryRowContext(
		ctx,
		insertLedger,
		ledger.ID,
		ledger.Name,
		ledger.Currency,
		ledger.CreatedBy,
		ledger.CreatedAt,
	).Scan(&lastId)
	if err != nil {
		return lastId, err
	}

	insertLedgerUser := `INSERT INTO ledger_users (ledger_id, user_id) VALUES ($1, $2)`
	_, err = tx.ExecContext(ctx, insertLedgerUser, ledger.ID, ledger.CreatedBy)
	if err != nil {
		return lastId, err
	}

	return lastId, tx.Commit()
}

func (r *repository) SaveExpense(ctx context.Context, expense Expense, splits []ExpenseSplit) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `INSERT INTO ledger_expenses (id, ledger_id, description, amount, paid_by, split_type, category, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
	_, err = tx.ExecContext(
		ctx,
		query,
		expense.ID,
		expense.LedgerID,
		expense.Description,
		expense.Amount,
		expense.PaidBy,
		expense.SplitType,
		expense.Category,
		expense.CreatedAt,
	)
	if err != nil {
		return err
	}

	for _, split := range splits {
		query = `INSERT INTO ledger_expense_splits (expense_id, user_id, amount) VALUES ($1, $2, $3)`
		_, err = tx.ExecContext(ctx, query, split.ExpenseID, split.UserID, split.Amount)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *repository) GetLedgerByID(ctx context.Context, ledgerID string) (*Ledger, error) {
	query := `SELECT id, name, currency, created_by, created_at FROM ledgers WHERE id = $1`

	var ledger Ledger
	err := r.db.QueryRowContext(ctx, query, ledgerID).Scan(
		&ledger.ID,
		&ledger.Name,
		&ledger.Currency,
		&ledger.CreatedBy,
		&ledger.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &ledger, nil
}

func (r *repository) GetLedgerMembers(ctx context.Context, ledgerID string) ([]LedgerUser, error) {
	query := `SELECT ledger_id, user_id, joined_at FROM ledger_users WHERE ledger_id = $1`

	rows, err := r.db.QueryContext(ctx, query, ledgerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []LedgerUser
	for rows.Next() {
		var member LedgerUser
		err := rows.Scan(&member.LedgerID, &member.UserID, &member.JoinedAt)
		if err != nil {
			return nil, err
		}
		members = append(members, member)
	}

	return members, rows.Err()
}

func (r *repository) GetRecentExpenses(ctx context.Context, ledgerID string, limit int) ([]Expense, error) {
	query := `SELECT id, ledger_id, description, amount, paid_by, split_type, category, created_at 
              FROM ledger_expenses 
              WHERE ledger_id = $1 
              ORDER BY created_at DESC 
              LIMIT $2`

	rows, err := r.db.QueryContext(ctx, query, ledgerID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var expenses []Expense
	for rows.Next() {
		var expense Expense
		var category sql.NullString
		err := rows.Scan(
			&expense.ID,
			&expense.LedgerID,
			&expense.Description,
			&expense.Amount,
			&expense.PaidBy,
			&expense.SplitType,
			&category,
			&expense.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		if category.Valid {
			expense.Category = category.String
		}
		expenses = append(expenses, expense)
	}

	return expenses, rows.Err()
}

func (r *repository) GetExpenseSplits(ctx context.Context, ledgerID string) ([]ExpenseSplit, error) {
	query := `SELECT es.expense_id, es.user_id, es.amount 
              FROM ledger_expense_splits es
              INNER JOIN ledger_expenses e ON es.expense_id = e.id
              WHERE e.ledger_id = $1`

	rows, err := r.db.QueryContext(ctx, query, ledgerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var splits []ExpenseSplit
	for rows.Next() {
		var split ExpenseSplit
		err := rows.Scan(&split.ExpenseID, &split.UserID, &split.Amount)
		if err != nil {
			return nil, err
		}
		splits = append(splits, split)
	}

	return splits, rows.Err()
}

func (r *repository) GetUserFirstLedger(ctx context.Context, userID string) (*Ledger, error) {
	query := `SELECT l.id, l.name, l.currency, l.created_by, l.created_at 
              FROM ledgers l
              INNER JOIN ledger_users lu ON l.id = lu.ledger_id
              WHERE lu.user_id = $1
              ORDER BY l.created_at ASC
              LIMIT 1`

	var ledger Ledger
	err := r.db.QueryRowContext(ctx, query, userID).Scan(
		&ledger.ID,
		&ledger.Name,
		&ledger.Currency,
		&ledger.CreatedBy,
		&ledger.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &ledger, nil
}
