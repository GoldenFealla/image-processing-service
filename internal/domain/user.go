package domain

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrUserNotFound     = errors.New("user not found")
	ErrIdentityNotFound = errors.New("identity not found")
)

type UserRepository interface {
	FindByID(ctx context.Context, id uuid.UUID) (*User, error)
	FindByEmail(ctx context.Context, email string) (*User, error)
	Create(ctx context.Context, user *User) error
}

type UserIdentityRepository interface {
	Create(ctx context.Context, userID uuid.UUID, Provider string, ProviderID string) error
	FindByProvider(ctx context.Context, Provider string, ProviderID string) (*UserIdentity, error)
}

type User struct {
	ID           uuid.UUID `db:"id"`
	Name         string    `db:"name"`
	Email        string    `db:"email"`
	Picture      string    `db:"picture"`
	PasswordHash string    `db:"password_hash"`
	CreatedAt    time.Time `db:"created_at"`
}

type UserIdentity struct {
	ID         uuid.UUID `db:"id"`
	UserID     uuid.UUID `db:"user_id"`
	Provider   string    `db:"provider"`
	ProviderID string    `db:"provider_id"`
	CreatedAt  time.Time `db:"created_at"`
}

type LoginForm struct {
	Email    string `db:"email"`
	Password string `db:"password"`
}

type RegisterForm struct {
	Name     string `db:"name"`
	Email    string `db:"email"`
	Password string `db:"password"`
}
