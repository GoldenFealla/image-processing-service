package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type ImageStore interface {
	GetOriginal(ctx context.Context, id uuid.UUID) ([]byte, error)
	SetOriginal(ctx context.Context, id uuid.UUID, data []byte) error
	GetTransformed(ctx context.Context, id uuid.UUID, opts TransformOptions) ([]byte, error)
	SetTransformed(ctx context.Context, id uuid.UUID, opts TransformOptions, data []byte) error
	DeleteOriginal(ctx context.Context, id uuid.UUID) error
	DeleteTransformed(ctx context.Context, id uuid.UUID) error
}

type SessionStore interface {
	IsRefreshTokenValid(ctx context.Context, token string) (uuid.UUID, error)
	SaveRefreshToken(ctx context.Context, userID uuid.UUID, token string, expiry time.Time) error
	RevokeRefreshToken(ctx context.Context, token string) error
}

type OAuthStateStore interface {
	SaveState(ctx context.Context, state string) error
	ValidateState(ctx context.Context, state string) (bool, error)
}
