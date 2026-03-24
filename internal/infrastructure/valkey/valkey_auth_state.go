package valkey

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type AuthStateStore struct {
	client *redis.Client
}

func NewAuthStateStore(cfg ValkeyConfig) (*AuthStateStore, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to session store: %w", err)
	}

	return &AuthStateStore{client: client}, nil
}

func (s *AuthStateStore) SaveState(ctx context.Context, state string) error {
	err := s.client.Set(ctx, stateKey(state), "1", 10*time.Minute).Err()
	if err != nil {
		return fmt.Errorf("failed to save oauth state: %w", err)
	}
	return nil
}

func (s *AuthStateStore) ValidateState(ctx context.Context, state string) (bool, error) {
	deleted, err := s.client.Del(ctx, stateKey(state)).Result()
	if errors.Is(err, redis.Nil) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to validate oauth state: %w", err)
	}
	return deleted == 1, nil
}

func stateKey(state string) string {
	return "oauth_state:" + state
}
