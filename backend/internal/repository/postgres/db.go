package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
)

// NewPool creates and validates a pgxpool.Pool from the given DSN.
// The pool is not connected until the first query; ParseConfig validates the DSN
// immediately so misconfiguration is caught at startup.
func NewPool(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("repo: parse dsn: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("repo: create pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("repo: ping database: %w", err)
	}

	return pool, nil
}

// NewDB wraps a pgxpool.Pool in a *sqlx.DB so that both pgx and sqlx helpers
// can be used against the same underlying pool.
func NewDB(pool *pgxpool.Pool) *sqlx.DB {
	db := stdlib.OpenDBFromPool(pool)
	return sqlx.NewDb(db, "pgx")
}
