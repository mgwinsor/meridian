// Package database owns PostgreSQL connection and migration lifecycle.
// Feature queries and repositories remain in their respective packages.
package database

import (
	"context"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

// URL defaults to the local development service in compose.yaml.
func URL() string {
	if value := os.Getenv("DATABASE_URL"); value != "" {
		return value
	}
	return "postgres://meridian:dev_password@127.0.0.1:5432/meridian?sslmode=disable"
}

func Open(ctx context.Context, url string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return pool, nil
}
