package ledger

import (
	"errors"
	"time"
)

type Member struct {
	ID   string
	Name string
}

type Ledger struct {
	Name      string
	Members   []Member
	Balances  map[string]int64
	Currency  string
	CreatedAt time.Time
}

func New(name string, members []Member, currency string, createdAt time.Time) (*Ledger, error) {
	if name == "" {
		return nil, errors.New("name can't be empty")
	}

	if len(members) == 0 {
		return nil, errors.New("ledger needs at least 1 member")
	}

	if currency == "" {
		return nil, errors.New("currency can't be empty")
	}

	now := createdAt
	if createdAt.UTC().IsZero() {
		now = time.Now().UTC()
	}

	return &Ledger{
		Name:      name,
		Members:   members,
		Balances:  make(map[string]int64, 0),
		Currency:  currency,
		CreatedAt: now,
	}, nil
}
