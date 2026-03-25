package config

import (
	"log"
	"os"
	"reflect"

	"github.com/GoldenFealla/image-processing-service/internal/infrastructure/pg"
)

type PostgresCloser struct {
	cfg    *pg.PostgresConfig
	closer []func()
}

func newPostgresCloser(cfg *pg.PostgresConfig) *PostgresCloser {
	return &PostgresCloser{cfg: cfg}
}

func (pc *PostgresCloser) Close() {
	for _, close := range pc.closer {
		close()
	}
}

func loadPostgresConfig() *pg.PostgresConfig {
	return &pg.PostgresConfig{
		Host:     os.Getenv("DB_HOST"),
		Port:     os.Getenv("DB_PORT"),
		User:     os.Getenv("DB_USER"),
		Password: os.Getenv("DB_PASSWORD"),
		DBName:   os.Getenv("DB_NAME"),
	}
}

func createPostgres[T interface{ Close() }](pc *PostgresCloser, factory func(cfg *pg.PostgresConfig) (T, error)) T {
	v, err := factory(pc.cfg)
	if err != nil {
		typeName := reflect.TypeFor[T]().Name()
		log.Fatalf("failed to initialize repository %s: %v", typeName, err)
	}
	pc.closer = append(pc.closer, v.Close)
	return v
}
