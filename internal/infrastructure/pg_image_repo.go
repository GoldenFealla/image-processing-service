package infrastructure

import (
	"context"
	"errors"
	"fmt"

	"github.com/GoldenFealla/image-processing-service/internal/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
}

func (c *PostgresConfig) DSN() string {
	// keyword/value format — no URL encoding needed
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		c.Host, c.Port, c.User, c.Password, c.DBName,
	)
}

type PostgresImageRepository struct {
	db *pgxpool.Pool
}

func NewPostgresImageRepository(cfg *PostgresConfig) (*PostgresImageRepository, error) {
	pool, err := pgxpool.New(context.Background(), cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("failed to connect to postgres: %w", err)
	}

	if err := pool.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to ping postgres: %w", err)
	}

	return &PostgresImageRepository{db: pool}, nil
}

func (pir *PostgresImageRepository) Save(ctx context.Context, image *domain.Image) error {
	_, err := pir.db.Exec(ctx,
		`INSERT INTO images (id, url) VALUES ($1, $2)`,
		image.ID, image.URL,
	)
	if err != nil {
		return fmt.Errorf("failed to save image: %w", err)
	}
	return nil
}
func (pir *PostgresImageRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Image, error) {
	row := pir.db.QueryRow(ctx,
		`SELECT id, url FROM images WHERE id = $1`, id,
	)

	image := &domain.Image{}
	err := row.Scan(&image.ID, &image.URL)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrImageNotFound
		}
		return nil, fmt.Errorf("failed to find image: %w", err)
	}
	return image, nil
}

func (pir *PostgresImageRepository) Update(ctx context.Context, image *domain.Image) error {
	_, err := pir.db.Exec(ctx,
		`UPDATE images SET url = $1, version = $2 WHERE id = $3`,
		image.URL, image.Version, image.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update image: %w", err)
	}
	return nil
}

func (pir *PostgresImageRepository) Close() {
	pir.db.Close()
}
