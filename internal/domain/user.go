package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrUserNotFound     = errors.New("user not found")
	ErrIdentityNotFound = errors.New("identity not found")
)

type User struct {
	ID           uuid.UUID `db:"id"`
	Username     string    `db:"username"`
	Email        string    `db:"email"`
	PasswordHash string    `db:"password_hash"`
	CreatedAt    time.Time `db:"created_at"`
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

type UserIdentity struct {
	ID         uuid.UUID `db:"id"`
	UserID     uuid.UUID `db:"user_id"`
	Provider   string    `db:"provider"`
	ProviderID string    `db:"provider_id"`
	CreatedAt  time.Time `db:"created_at"`
}
