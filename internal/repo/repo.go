package repo

import (
	"context"
	"fmt"
	"gRPC/internal/config"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pkg/errors"
)

const (
	createUserQuery = `
		INSERT INTO users (username, hashed_password, created_at, updated_at)
		VALUES ($1, $2, NOW(), NOW());
	`

	getUserByUsernameQuery = `
		SELECT id, username, hashed_password, created_at, updated_at
		FROM users
		WHERE username = $1;
	`
)

type repository struct {
	pool *pgxpool.Pool
}

type Repository interface {
	CreateUser(ctx context.Context, user *User) error
	Login(ctx context.Context, username string) (*User, error)
}

func NewRepository(ctx context.Context, cfg config.PostgreSQL) (Repository, error) {
	connString := fmt.Sprintf(
		`user=%s password=%s host=%s port=%d dbname=%s sslmode=%s 
        pool_max_conns=%d pool_max_conn_lifetime=%s pool_max_conn_idle_time=%s`,
		cfg.User,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.Name,
		cfg.SSLMode,
		cfg.PoolMaxConns,
		cfg.PoolMaxConnLifetime.String(),
		cfg.PoolMaxConnIdleTime.String(),
	)

	config, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse PostgreSQL config")
	}

	config.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeCacheDescribe

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create PostgreSQL connection pool")
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, errors.Wrap(err, "the connection doesn't ping")
	}

	return &repository{pool}, nil
}

func (r *repository) CreateUser(ctx context.Context, user *User) error {
	_, err := r.pool.Exec(ctx, createUserQuery, user.Username, user.HashedPassword)
	if err != nil {
		return errors.Wrap(err, "failed to insert user")
	}
	return nil
}

func (r *repository) Login(ctx context.Context, username string) (*User, error) {
	var user User
	err := r.pool.QueryRow(ctx, getUserByUsernameQuery, username).Scan(
		&user.ID,
		&user.Username,
		&user.HashedPassword,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get user credentials")
	}
	return &user, nil
}
