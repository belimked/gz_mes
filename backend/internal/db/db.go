// Package db opens the PostgreSQL pool and runs migrations.
package db

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"github.com/belimked/gz_mes/backend/db/migrations"
)

// Open connects to PostgreSQL and verifies the connection.
func Open(ctx context.Context, url string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping db: %w", err)
	}
	return pool, nil
}

// Migrate applies all pending migrations.
func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	return withGoose(pool, func(sqlDB *sql.DB) error { return goose.UpContext(ctx, sqlDB, ".") })
}

// Reset rolls back all migrations (used by tests to verify down migrations).
func Reset(ctx context.Context, pool *pgxpool.Pool) error {
	return withGoose(pool, func(sqlDB *sql.DB) error { return goose.ResetContext(ctx, sqlDB, ".") })
}

func withGoose(pool *pgxpool.Pool, fn func(*sql.DB) error) error {
	sqlDB := stdlib.OpenDBFromPool(pool)
	defer sqlDB.Close()
	goose.SetBaseFS(migrations.FS)
	goose.SetLogger(goose.NopLogger())
	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}
	return fn(sqlDB)
}
