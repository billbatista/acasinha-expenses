package user

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrEmailExists   = errors.New("email already exists")
	ErrInvalidEmail  = errors.New("invalid email format")
	ErrBlankPassword = errors.New("password can't be blank")
)

type repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *repository {
	return &repository{db: db}
}

func (r *repository) Register(ctx context.Context, email, password string) (*User, error) {
	if email == "" {
		return nil, ErrInvalidEmail
	}

	if password == "" {
		return nil, ErrBlankPassword
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hashing password: %w", err)
	}

	user := &User{
		ID:           uuid.New(),
		Email:        email,
		PasswordHash: string(hashedPassword),
		CreatedAt:    time.Now().UTC(),
	}

	query := `INSERT INTO users (id, email, password_hash, created_at) VALUES ($1, $2, $3, $4)`
	_, err = r.db.ExecContext(ctx, query, user.ID, user.Email, user.PasswordHash, user.CreatedAt)
	if err != nil {
		// check for unique constraint error
		// return nil, ErrEmailExists
		return nil, fmt.Errorf("inserting user: %w", err)
	}

	return user, nil
}

func (r *repository) GetByEmail(ctx context.Context, email string) (*User, error) {
	query := `SELECT id, COALESCE(name, ''), email, password_hash, created_at, avatar FROM users WHERE email = $1`

	var user User
	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.PasswordHash,
		&user.CreatedAt,
		&user.Avatar,
	)
	if err != nil && err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("querying user: %w", err)
	}

	return &user, nil
}

func (r *repository) GetByID(ctx context.Context, id uuid.UUID) (*User, error) {
	query := `SELECT id, COALESCE(name, ''), email, password_hash, created_at, avatar FROM users WHERE id = $1`

	var user User
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.PasswordHash,
		&user.CreatedAt,
		&user.Avatar,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &user, nil
}

func (r *repository) UpdateName(ctx context.Context, userID uuid.UUID, name string) error {
	query := `UPDATE users SET name = $1 WHERE id = $2`
	_, err := r.db.ExecContext(ctx, query, name, userID)
	return err
}

func (r *repository) VerifyPassword(hashedPassword, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}

func (r *repository) UpdateAvatar(ctx context.Context, img []byte, userId uuid.UUID) error {
	query := `UPDATE users SET avatar = $1 WHERE id = $2`
	_, err := r.db.ExecContext(ctx, query, img, userId)
	return err
}
