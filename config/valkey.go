package config

import (
	"log"
	"os"
	"reflect"
	"time"

	"github.com/GoldenFealla/image-processing-service/internal/infrastructure/valkey"
)

func loadValkeyConfig(db int, ttl time.Duration) valkey.ValkeyConfig {
	return valkey.ValkeyConfig{
		Addr:     os.Getenv("VALKEY_ADDR"),
		Password: os.Getenv("VALKEY_PASSWORD"),
		DB:       db,
		TTL:      ttl,
	}
}

func createValkey[T any](cfg valkey.ValkeyConfig, factory func(cfg valkey.ValkeyConfig) (T, error)) T {
	v, err := factory(cfg)
	if err != nil {
		typeName := reflect.TypeFor[T]().Name()
		log.Fatalf("failed to initialize valkey %s: %v", typeName, err)
	}
	return v
}
