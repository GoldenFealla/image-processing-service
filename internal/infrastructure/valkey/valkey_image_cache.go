// infrastructure/image_cache.go
package valkey

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"time"

	"github.com/GoldenFealla/image-processing-service/internal/domain"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type ImageCache struct {
	client *redis.Client
	ttl    time.Duration
}

func NewImageCache(cfg ValkeyConfig) (*ImageCache, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to valkey: %w", err)
	}

	ttl := cfg.TTL
	if ttl == 0 {
		ttl = 24 * time.Hour
	}

	return &ImageCache{client: client, ttl: ttl}, nil
}

func (c *ImageCache) GetOriginal(ctx context.Context, id uuid.UUID) ([]byte, error) {
	data, err := c.client.Get(ctx, c.originalKey(id)).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // cache miss
		}
		return nil, fmt.Errorf("failed to get cached image: %w", err)
	}
	return data, nil
}

func (c *ImageCache) SetOriginal(ctx context.Context, id uuid.UUID, data []byte) error {
	if err := c.client.Set(ctx, c.originalKey(id), data, c.ttl).Err(); err != nil {
		return fmt.Errorf("failed to cache image: %w", err)
	}
	return nil
}

func (c *ImageCache) GetTransformed(ctx context.Context, id uuid.UUID, opts domain.TransformOptions) ([]byte, error) {
	data, err := c.client.Get(ctx, c.transformKey(id, opts)).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // cache miss
		}
		return nil, fmt.Errorf("failed to get transformed cache: %w", err)
	}
	return data, nil
}

func (c *ImageCache) SetTransformed(ctx context.Context, id uuid.UUID, opts domain.TransformOptions, data []byte) error {
	if err := c.client.Set(ctx, c.transformKey(id, opts), data, c.ttl).Err(); err != nil {
		return fmt.Errorf("failed to cache transformed image: %w", err)
	}
	return nil
}

func (c *ImageCache) Close() error {
	return c.client.Close()
}

func (c *ImageCache) originalKey(id uuid.UUID) string {
	return fmt.Sprintf("image:original:%s", id.String())
}

func (c *ImageCache) transformKey(id uuid.UUID, opts domain.TransformOptions) string {
	h := fnv.New64a()
	b, _ := json.Marshal(opts)
	h.Write(b)
	return fmt.Sprintf("image:transform:%s:%d", id.String(), h.Sum64())
}

func (c *ImageCache) DeleteOriginal(ctx context.Context, id uuid.UUID) error {
	if err := c.client.Del(ctx, c.originalKey(id)).Err(); err != nil {
		return fmt.Errorf("failed to delete cached image: %w", err)
	}
	return nil
}

func (c *ImageCache) DeleteTransformed(ctx context.Context, id uuid.UUID) error {
	pattern := fmt.Sprintf("image:transform:%s:*", id.String())
	keys, err := c.client.Keys(ctx, pattern).Result()
	if err != nil {
		return fmt.Errorf("failed to scan transformed cache keys: %w", err)
	}
	if len(keys) == 0 {
		return nil
	}
	if err := c.client.Del(ctx, keys...).Err(); err != nil {
		return fmt.Errorf("failed to delete transformed cache: %w", err)
	}
	return nil
}
