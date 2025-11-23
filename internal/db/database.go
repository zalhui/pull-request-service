package db

import (
	"context"
	"fmt"

	"github.com/zalhui/pull-request-service/internal/config"

	"github.com/jackc/pgx/v5/pgxpool"
)

func NewConnection(ctx context.Context, cfg *config.Config) (*pgxpool.Pool, error) {
	connStr := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=disable",
		cfg.DBUser, cfg.DBPassword, cfg.DBHost, cfg.DBPort, cfg.DBName,
	)

	config, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse connection string: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return pool, nil
}
