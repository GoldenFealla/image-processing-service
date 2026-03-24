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

type PostgresUserIdentityRepository struct {
	db *pgxpool.Pool
}

func NewPostgresUserIdentityRepository(cfg *PostgresConfig) (*PostgresUserIdentityRepository, error) {
	pool, err := pgxpool.New(context.Background(), cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("failed to connect to postgres: %w", err)
	}

	if err := pool.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to ping postgres: %w", err)
	}

	return &PostgresUserIdentityRepository{db: pool}, nil
}

func (r *PostgresUserIdentityRepository) Close() {
	r.db.Close()
}

func (r *PostgresUserIdentityRepository) FindByProvider(ctx context.Context, Provider string, ProviderID string) (*domain.UserIdentity, error) {
	query := `
		SELECT id, user_id, provider, provider_id, created_at
		FROM user_identities WHERE provider = $1 AND provider_id = $2
	`
	rows, err := r.db.Query(ctx, query, Provider, ProviderID)
	if err != nil {
		return nil, errors.Join(domain.ErrIdentityNotFound, err)
	}
	userIdentity, err := pgx.CollectOneRow(rows, pgx.RowToAddrOfStructByName[domain.UserIdentity])
	if err != nil {
		return nil, errors.Join(domain.ErrIdentityNotFound, err)
	}
	return userIdentity, nil
}

func (r *PostgresUserIdentityRepository) Create(ctx context.Context, userID uuid.UUID, Provider string, ProviderID string) error {
	query := `
		INSERT INTO user_identities (user_id, provider, provider_id, created_at)
		VALUES ($1, $2, $3, NOW())
	`
	_, err := r.db.Exec(ctx, query, userID, Provider, ProviderID)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}
	return nil
}
