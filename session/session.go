package session

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrInvalidSession = errors.New("invalid session")
	ErrExpiredSession = errors.New("session expired")
)

const (
	sessionDuration = 7 * 24 * time.Hour
	CookieName      = "session_token"
)

type Session struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	Token     string
	ExpiresAt time.Time
	CreatedAt time.Time
}

type Repository interface {
	Create(ctx context.Context, userID uuid.UUID) (*Session, error)
	GetByToken(ctx context.Context, token string) (*Session, error)
	Delete(ctx context.Context, token string) error
	DeleteByUserID(ctx context.Context, userID uuid.UUID) error
}
