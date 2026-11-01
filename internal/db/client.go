package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"school-attendance/migrations"
)

type DB struct {
	Pool    *pgxpool.Pool
	Queries *Queries
}

// Connect initializes the database connection, runs migrations via goose,
// and returns a DB instance with pgxpool and sqlc Queries.
func Connect(ctx context.Context, databaseURL string) (*DB, error) {
	// 1. Run migrations using standard sql.DB
	sqlDB, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database for migrations: %w", err)
	}
	defer sqlDB.Close()

	if err := sqlDB.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	if err := migrations.RunMigrations(sqlDB); err != nil {
		return nil, fmt.Errorf("failed to apply migrations: %w", err)
	}

	// 2. Setup pgxpool
	poolConfig, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse pgxpool config: %w", err)
	}
	poolConfig.MaxConns = 25
	poolConfig.MinConns = 5
	poolConfig.MaxConnLifetime = 1 * time.Hour
	poolConfig.MaxConnIdleTime = 30 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping pgxpool: %w", err)
	}

	return &DB{
		Pool:    pool,
		Queries: New(pool),
	}, nil
}

