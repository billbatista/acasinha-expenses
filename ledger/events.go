package ledger

import "time"

type LedgerCreatedEvent struct {
	Name      string
	Members   []Member
	Currency  string
	CreatedAt time.Time
}

type ExpenseAddedEvent struct {
	PaidByUserId string
	AmountCents  int64  // Total amount in cents
	Description  string // What the expense was for
	Category     string // e.g., "groceries", "utilities", "rent"
	Date         time.Time
	ShareType    int // 0=equal split, 1=percentage, 2=exact amounts
	Currency     string
	Splits       []Split // How the expense is divided among members
}

type Split struct {
	UserID      string
	Percentage  float64 // 0-100
	AmountCents int64   // Amount in cents this person owes for this expense
}

type ManuallyCorrectedEvent struct {
	MadeByUserId string
	Reason       string
}
