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

	createTokensQuery = `
		INSERT INTO users_tokens (user_id, access_token, refresh_token)
		VALUES ($1, $2, $3)
		RETURNING id;
	`

	getPasswordQuery = `
		SELECT hashed_password
		FROM users
		WHERE username = $1;
	`

	updatePasswordQuery = `
		UPDATE users 
		SET hashed_password = $1
		WHERE username = $2
		RETURNING id;
	`

	getRefreshTokenQuery = `
		SELECT refresh_token
		FROM users_tokens
		WHERE user_id = $1;
	`

	updateRefreshTokenQuery = `
		UPDATE users_tokens
		SET refresh_token = $1
		WHERE user_id = $2
		RETURNING user_id;
	`

	deleteRefreshTokenQuery = `
		DELETE FROM users_tokens
		WHERE user_id = $1
		RETURNING user_id;
	`
)

type repository struct {
	pool *pgxpool.Pool
}

type Repository interface {
	CreateUser(ctx context.Context, user *User) (int, error)
	GetUser(ctx context.Context, username string) (*User, error)
	CreateTokens(ctx context.Context, users_tokens *User_Tokens) (int, error)
	GetPassword(ctx context.Context, username string) (string, error)
	UpdatePassword(ctx context.Context, data UpdatePasswordData) error
	GetRefreshToken(ctx context.Context, user_id int64) (string, error)
	NewRefreshToken(ctx context.Context, params NewRefreshTokenParams) error
	DeleteRefreshToken(ctx context.Context, user_id int64) error
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

func (r *repository) GetPassword(ctx context.Context, username string) (string, error) {
	var password string

	err := r.pool.QueryRow(ctx, getPasswordQuery, username).Scan(&password)

	if err != nil {
		if err == sql.ErrNoRows {
			return "", errors.New("User not exist")
		}
		return "", err
	}
	return password, nil
}

func (r *repository) UpdatePassword(ctx context.Context, data UpdatePasswordData) error {
	var id int
	err := r.pool.QueryRow(ctx, updatePasswordQuery, data.NewPassword, data.Username).Scan(&id)
	if err != nil {
		return err
	}

	return nil
}

func (r *repository) CreateTokens(ctx context.Context, users_tokens *User_Tokens) (int, error) {
	var id int
	err := r.pool.QueryRow(ctx, createTokensQuery, users_tokens.User_ID, users_tokens.AccessToken, users_tokens.RefreshToken).Scan(&id)
	if err != nil {
		return 0, errors.Wrap(err, "failed to insert user_tokens")
	}
	return id, nil
}

func (r *repository) GetRefreshToken(ctx context.Context, user_id int64) (string, error) {
	var refreshToken string
	err := r.pool.QueryRow(ctx, getRefreshTokenQuery, user_id).Scan(&refreshToken)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", errors.New("User not exist")
		}
		return "", err
	}
	return refreshToken, nil
}

func (r *repository) NewRefreshToken(ctx context.Context, params NewRefreshTokenParams) error {
	var id int
	err := r.pool.QueryRow(ctx, updateRefreshTokenQuery, params.Token, params.UserID).Scan(&id)
	if err != nil {
		return err
	}
	return nil
}

func (r *repository) DeleteRefreshToken(ctx context.Context, user_id int64) error {
	var id int
	err := r.pool.QueryRow(ctx, deleteRefreshTokenQuery, user_id).Scan(&id)
	if err != nil {
		return err
	}
	return nil
}
