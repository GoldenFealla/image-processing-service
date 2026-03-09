package domain

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID           uuid.UUID
	Email        string
	PasswordHash string

	Provider   string
	ProviderID string

	CreatedAt time.Time
}
