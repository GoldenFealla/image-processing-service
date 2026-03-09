package valkey

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type SessionStore struct {
	client *redis.Client
}

func NewSessionStore(cfg ValkeyConfig) (*SessionStore, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to session store: %w", err)
	}

	return &SessionStore{client: client}, nil
}

func (s *SessionStore) Close() {
	s.client.Close()
}

func (s *SessionStore) SaveRefreshToken(ctx context.Context, userID uuid.UUID, token string, expiry time.Time) error {
	ttl := time.Until(expiry)
	err := s.client.Set(ctx, refreshKey(token), userID.String(), ttl).Err()
	if err != nil {
		return fmt.Errorf("failed to save refresh token: %w", err)
	}
	return nil
}

func (s *SessionStore) IsRefreshTokenValid(ctx context.Context, token string) (uuid.UUID, error) {
	val, err := s.client.Get(ctx, refreshKey(token)).Result()
	if errors.Is(err, redis.Nil) {
		return uuid.Nil, errors.New("refresh token not found or expired")
	}
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to get refresh token: %w", err)
	}

	userID, err := uuid.Parse(val)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid user id in session: %w", err)
	}

	return userID, nil
}

func (s *SessionStore) RevokeRefreshToken(ctx context.Context, token string) error {
	err := s.client.Del(ctx, refreshKey(token)).Err()
	if err != nil {
		return fmt.Errorf("failed to revoke refresh token: %w", err)
	}
	return nil
}

func refreshKey(token string) string {
	return "refresh:" + token
}
