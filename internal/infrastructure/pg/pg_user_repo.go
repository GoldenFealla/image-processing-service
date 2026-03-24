package pg

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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

func (r *PostgresUserRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	query := `
		SELECT id, username, email, password_hash, created_at
		FROM users WHERE id = $1
	`
	rows, err := r.db.Query(ctx, query, id)
	if err != nil {
		return nil, errors.Join(domain.ErrUserNotFound, err)
	}
	user, err := pgx.CollectOneRow(rows, pgx.RowToAddrOfStructByName[domain.User])
	if err != nil {
		return nil, errors.Join(domain.ErrUserNotFound, err)
	}
	return user, nil
}

func (r *PostgresUserRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `
		SELECT id, username, email, password_hash, created_at
		FROM users WHERE email = $1
	`
	rows, err := r.db.Query(ctx, query, email)
	if err != nil {
		return nil, errors.Join(domain.ErrUserNotFound, err)
	}
	user, err := pgx.CollectOneRow(rows, pgx.RowToAddrOfStructByName[domain.User])
	if err != nil {
		return nil, errors.Join(domain.ErrUserNotFound, err)
	}
	return user, nil
}

func (r *PostgresUserRepository) FindByUsernameOrEmail(ctx context.Context, usernameOrEmail string) (*domain.User, error) {
	query := `
		SELECT id, username, email, password_hash, created_at
		FROM users WHERE username = $1 OR email = $2
	`
	rows, err := r.db.Query(ctx, query, usernameOrEmail, usernameOrEmail)
	if err != nil {
		return nil, errors.Join(domain.ErrUserNotFound, err)
	}
	user, err := pgx.CollectOneRow(rows, pgx.RowToAddrOfStructByName[domain.User])
	if err != nil {
		return nil, errors.Join(domain.ErrUserNotFound, err)
	}
	return user, nil
}

func (r *PostgresUserRepository) Create(ctx context.Context, user *domain.User) error {
	user.ID = uuid.Must(uuid.NewV7())

	query := `
		INSERT INTO users (id, username, email, password_hash, created_at)
		VALUES ($1, $2, $3, $4, NOW())
	`
	_, err := r.db.Exec(ctx, query, user.ID, user.Username, user.Email, user.PasswordHash)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}
	return nil
}
