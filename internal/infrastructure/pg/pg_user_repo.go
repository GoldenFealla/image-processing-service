package pg

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/GoldenFealla/image-processing-service/internal/domain"
)

type PostgresUserRepository struct {
	db *pgxpool.Pool
}

func NewPostgresUserRepository(cfg *PostgresConfig) (*PostgresUserRepository, error) {
	pool, err := pgxpool.New(context.Background(), cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("failed to connect to postgres: %w", err)
	}

	if err := pool.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to ping postgres: %w", err)
	}

	return &PostgresUserRepository{db: pool}, nil
}

func (r *PostgresUserRepository) Close() {
	r.db.Close()
}

func (r *PostgresUserRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	user := &domain.User{}
	err := r.db.QueryRow(ctx, `
		SELECT id, email, password_hash, provider, provider_id, created_at
		FROM users WHERE email = $1
	`, email).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.Provider, &user.ProviderID, &user.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}
	return user, nil
}

func (r *PostgresUserRepository) FindByProviderID(ctx context.Context, provider, providerID string) (*domain.User, error) {
	user := &domain.User{}
	err := r.db.QueryRow(ctx, `
		SELECT id, email, password_hash, provider, provider_id, created_at
		FROM users WHERE provider = $1 AND provider_id = $2
	`, provider, providerID).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.Provider, &user.ProviderID, &user.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}
	return user, nil
}

func (r *PostgresUserRepository) Create(ctx context.Context, user *domain.User) error {
	user.ID = uuid.Must(uuid.NewV7())
	user.CreatedAt = time.Now()

	_, err := r.db.Exec(ctx, `
		INSERT INTO users (id, email, password_hash, provider, provider_id, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, user.ID, user.Email, user.PasswordHash, user.Provider, user.ProviderID, user.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}
	return nil
}
