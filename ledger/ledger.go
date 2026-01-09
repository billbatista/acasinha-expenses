package ledger

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

type SplitType string

const (
	SplitTypeEqual SplitType = "equal"
	// TODO: SplitTypePercentage, SplitTypeExact
)

type Ledger struct {
	ID        uuid.UUID `json:"id,omitempty"`
	Name      string    `json:"name,omitempty"`
	Currency  string    `json:"currency,omitempty"`
	CreatedBy uuid.UUID `json:"created_by,omitempty"`
	CreatedAt time.Time `json:"created_at,omitempty"`
}

type Expense struct {
	ID          uuid.UUID `json:"id,omitempty"`
	LedgerID    uuid.UUID `json:"ledger_id,omitempty"`
	Description string    `json:"description,omitempty"`
	Amount      int64     `json:"amount,omitempty"` // Amount in cents
	PaidBy      uuid.UUID `json:"paid_by,omitempty"`
	SplitType   SplitType `json:"split_type,omitempty"`
	Category    string    `json:"category,omitempty"`
	CreatedAt   time.Time `json:"created_at,omitempty"`
}

type ExpenseSplit struct {
	ExpenseID uuid.UUID `json:"expense_id,omitempty"`
	UserID    uuid.UUID `json:"user_id,omitempty"`
	Amount    int64     `json:"amount,omitempty"` // Amount owed in cents
}

type LedgerUser struct {
	LedgerID uuid.UUID `json:"ledger_id,omitempty"`
	UserID   uuid.UUID `json:"user_id,omitempty"`
	JoinedAt time.Time `json:"joined_at,omitempty"`
}

// Balance represents a user's net balance in a ledger
// Calculated on-the-fly from expenses
type Balance struct {
	UserID uuid.UUID
	Amount int64 // Positive = owed money, Negative = owes money
}

var (
	ErrEmptyName        = errors.New("name can't be empty")
	ErrEmptyCurrency    = errors.New("currency can't be empty")
	ErrInvalidAmount    = errors.New("amount must be positive")
	ErrEmptyDescription = errors.New("description can't be empty")
)

func NewLedger(name string, currency string, createdBy uuid.UUID) (Ledger, error) {
	if name == "" {
		return Ledger{}, ErrEmptyName
	}

	if currency == "" {
		return Ledger{}, ErrEmptyCurrency
	}

	now := time.Now().UTC()

	return Ledger{
		ID:        uuid.New(),
		Name:      name,
		Currency:  currency,
		CreatedBy: createdBy,
		CreatedAt: now,
	}, nil
}

func NewExpense(ledgerID uuid.UUID, description string, amount int64, paidBy uuid.UUID, splitType SplitType, category string, memberIDs []uuid.UUID) (*Expense, []ExpenseSplit, error) {
	if description == "" {
		return nil, nil, ErrEmptyDescription
	}

	if amount <= 0 {
		return nil, nil, ErrInvalidAmount
	}

	expense := &Expense{
		ID:          uuid.New(),
		LedgerID:    ledgerID,
		Description: description,
		Amount:      amount,
		PaidBy:      paidBy,
		SplitType:   splitType,
		Category:    category,
		CreatedAt:   time.Now().UTC(),
	}

	splits, err := CalculateSplits(expense.ID, amount, splitType, memberIDs)
	if err != nil {
		return nil, nil, err
	}

	return expense, splits, nil
}

func CalculateSplits(expenseID uuid.UUID, amount int64, splitType SplitType, memberIDs []uuid.UUID) ([]ExpenseSplit, error) {
	numMembers := int64(len(memberIDs))
	if numMembers == 0 {
		return nil, errors.New("no members to split expense")
	}

	splits := make([]ExpenseSplit, 0, numMembers)

	switch splitType {
	case SplitTypeEqual:
		baseAmount := amount / numMembers
		remainder := amount % numMembers

		for i, userID := range memberIDs {
			share := baseAmount
			// Distribute remainder to first few members
			if int64(i) < remainder {
				share++
			}
			splits = append(splits, ExpenseSplit{
				ExpenseID: expenseID,
				UserID:    userID,
				Amount:    share,
			})
		}
		return splits, nil

	default:
		return nil, errors.New("unsupported split type")
	}
}

// CalculateBalances computes net balances for all users from expenses and their splits
func CalculateBalances(expenses []Expense, splits []ExpenseSplit, memberIDs []uuid.UUID) map[uuid.UUID]int64 {
	balances := make(map[uuid.UUID]int64)

	// Initialize all members with 0 balance
	for _, userID := range memberIDs {
		balances[userID] = 0
	}

	// Credit the payer for each expense
	for _, expense := range expenses {
		balances[expense.PaidBy] += expense.Amount
	}

	// Debit each member for their share
	for _, split := range splits {
		balances[split.UserID] -= split.Amount
	}

	return balances
}
