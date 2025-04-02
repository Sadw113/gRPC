package repo

import (
	"context"
	"database/sql"
	"fmt"
	"gRPC/internal/config"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pkg/errors"
)

const (
	createUserQuery = `
		INSERT INTO users (username, hashed_password, created_at, updated_at)
		VALUES ($1, $2, NOW(), NOW())
		RETURNING id;
	`

	getUserByUsernameQuery = `
		SELECT id, username, hashed_password, created_at, updated_at
		FROM users
		WHERE username = $1;
	`

	getPasswordQuery = `
		SELECT hashed_password
		FROM users
		WHERE id = $1;
	`

	updatePasswordQuery = `
		UPDATE users 
		SET hashed_password = $1
		WHERE id = $2;
	`
)

type repository struct {
	pool *pgxpool.Pool
}

type Repository interface {
	CreateUser(ctx context.Context, user *User) (int, error)
	GetUser(ctx context.Context, username string) (*User, error)
	GetPassword(ctx context.Context, userID int64) (string, error)
	UpdatePassword(ctx context.Context, newPassword string, userid int64) error
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

func (r *repository) CreateUser(ctx context.Context, user *User) (int, error) {
	var id int
	err := r.pool.QueryRow(ctx, createUserQuery, user.Username, user.HashedPassword).Scan(&id)
	if err != nil {
		return 0, errors.Wrap(err, "failed to insert user")
	}
	return id, nil
}

func (r *repository) GetUser(ctx context.Context, username string) (*User, error) {
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

func (r *repository) GetPassword(ctx context.Context, userID int64) (string, error) {
	var password string

	err := r.pool.QueryRow(ctx, getPasswordQuery, userID).Scan(&password)

	if err != nil {
		if err == sql.ErrNoRows {
			return "", errors.New("User not exist")
		}

		return "", err
	}

	return password, nil
}

func (r *repository) UpdatePassword(ctx context.Context, newPassword string, userid int64) error {
	_, err := r.pool.Exec(ctx, updatePasswordQuery, newPassword, userid)
	if err != nil {
		return err
	}

	return nil
}
