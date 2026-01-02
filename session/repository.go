package session

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"time"

	"github.com/google/uuid"
)

type repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *repository {
	return &repository{db: db}
}

func (r *repository) Create(ctx context.Context, userID uuid.UUID) (*Session, error) {
	token, err := generateSecureToken()
	if err != nil {
		return nil, err
	}

	session := &Session{
		ID:        uuid.New(),
		UserID:    userID,
		Token:     token,
		ExpiresAt: time.Now().Add(sessionDuration),
		CreatedAt: time.Now(),
	}

	query := `
        INSERT INTO sessions (id, user_id, token, expires_at, created_at)
        VALUES ($1, $2, $3, $4, $5)
    `

	_, err = r.db.ExecContext(ctx, query,
		session.ID,
		session.UserID,
		session.Token,
		session.ExpiresAt,
		session.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	return session, nil
}

// GetByToken retrieves a session by token and validates it's not expired
func (r *repository) GetByToken(ctx context.Context, token string) (*Session, error) {
	var session Session

	query := `
        SELECT id, user_id, token, expires_at, created_at
        FROM sessions
        WHERE token = $1
    `

	err := r.db.QueryRowContext(ctx, query, token).Scan(
		&session.ID,
		&session.UserID,
		&session.Token,
		&session.ExpiresAt,
		&session.CreatedAt,
	)
	if err != nil && err == sql.ErrNoRows {
		return nil, ErrInvalidSession
	}
	if err != nil {
		return nil, err
	}

	if time.Now().After(session.ExpiresAt) {
		return nil, ErrExpiredSession
	}

	return &session, nil
}

// Delete removes a session (logout)
func (r *repository) Delete(ctx context.Context, token string) error {
	query := `DELETE FROM sessions WHERE token = $1`
	_, err := r.db.ExecContext(ctx, query, token)
	return err
}

// DeleteByUserID removes all sessions for a user
func (r *repository) DeleteByUserID(ctx context.Context, userID uuid.UUID) error {
	query := `DELETE FROM sessions WHERE user_id = $1`
	_, err := r.db.ExecContext(ctx, query, userID)
	return err
}

func generateSecureToken() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}
