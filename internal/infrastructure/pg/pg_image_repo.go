package pg

import (
	"context"
	"errors"
	"fmt"

	"github.com/GoldenFealla/image-processing-service/internal/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

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

func (pir *PostgresImageRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Image, error) {
	row := pir.db.QueryRow(ctx,
		`SELECT id, url, version, owner_id FROM images WHERE id = $1`, id,
	)

	image := &domain.Image{}
	err := row.Scan(&image.ID, &image.URL, &image.Version, &image.OwnerID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrImageNotFound
		}
		return nil, fmt.Errorf("failed to find image: %w", err)
	}
	return image, nil
}

func (pir *PostgresImageRepository) FindListByOwnerID(ctx context.Context, userID uuid.UUID) ([]*domain.Image, error) {
	rows, err := pir.db.Query(ctx,
		`SELECT id, url, version, owner_id FROM images WHERE owner_id = $1`, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query images: %w", err)
	}
	defer rows.Close()

	images, err := pgx.CollectRows(rows, pgx.RowToAddrOfStructByName[domain.Image])
	if err != nil {
		return nil, fmt.Errorf("failed to collect images: %w", err)
	}

	if len(images) == 0 {
		return nil, domain.ErrImageNotFound
	}

	return images, nil
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

func (pir *PostgresImageRepository) Save(ctx context.Context, image *domain.Image) error {
	_, err := pir.db.Exec(ctx,
		`INSERT INTO images (id, url, version, owner_id) VALUES ($1, $2, $3, $4)`,
		image.ID, image.URL, image.Version, image.OwnerID,
	)
	if err != nil {
		return fmt.Errorf("failed to save image: %w", err)
	}
	return nil
}

func (pir *PostgresImageRepository) Close() {
	pir.db.Close()
}
