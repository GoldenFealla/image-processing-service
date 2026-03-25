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
	FindByUsernameOrEmail(ctx context.Context, usernameOrEmail string) (*User, error)
	Create(ctx context.Context, user *User) error
}

type UserIdentityRepository interface {
	Create(ctx context.Context, userID uuid.UUID, Provider string, ProviderID string) error
	FindByProvider(ctx context.Context, Provider string, ProviderID string) (*UserIdentity, error)
}

type User struct {
	ID           uuid.UUID `db:"id"`
	Username     string    `db:"username"`
	Email        string    `db:"email"`
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
	UsernameOrEmail string `db:"username_or_password"`
	Password        string `db:"password"`
}

type RegisterForm struct {
	Username string `db:"username"`
	Email    string `db:"email"`
	Password string `db:"password"`
}
