package user

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID           uuid.UUID `json:"id"`
	Name         string    `json:"name"`
	Email        string    `json:"email"`
	Avatar       []byte    `json:"avatar"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}

type Repository interface {
	Register(ctx context.Context, email, password string) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	VerifyPassword(hashedPassword, password string) error
	UpdateName(ctx context.Context, userID uuid.UUID, name string) error
	UpdateAvatar(ctx context.Context, img []byte) error
}
