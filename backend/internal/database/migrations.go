package database

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"io/fs"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/lock"
)

//go:embed migrations/*.sql
var migrations embed.FS

// Create runs an idempotent goose Go migration against the maintenance database.
// CREATE DATABASE must run outside a transaction. Bootstrap is unversioned so it
// also works with a database already initialized by Docker's POSTGRES_DB setting.
func Create(ctx context.Context, url string) error {
	config, err := pgx.ParseConfig(url)
	if err != nil {
		return err
	}
	name := config.Database
	if name == "" {
		return errors.New("DATABASE_URL must specify a database name")
	}
	config.Database = "postgres"
	db := stdlib.OpenDB(*config)
	defer db.Close()
	locker, err := lock.NewPostgresSessionLocker()
	if err != nil {
		return err
	}

	migration := goose.NewGoMigration(1, &goose.GoFunc{
		RunDB: func(ctx context.Context, db *sql.DB) error {
			var exists bool
			if err := db.QueryRowContext(ctx, "SELECT EXISTS (SELECT 1 FROM pg_database WHERE datname = $1)", name).Scan(&exists); err != nil {
				return err
			}
			if exists {
				return nil
			}
			_, err := db.ExecContext(ctx, "CREATE DATABASE "+pgx.Identifier{name}.Sanitize())
			// Another initializer may have created the database after our lookup.
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "42P04" {
				return nil
			}
			return err
		},
	}, nil)
	provider, err := goose.NewProvider(goose.DialectPostgres, db, nil,
		goose.WithDisableGlobalRegistry(true), goose.WithDisableVersioning(true),
		goose.WithGoMigrations(migration), goose.WithSessionLocker(locker))
	if err != nil {
		return err
	}
	_, err = provider.Up(ctx)
	return err
}

// RequireCurrentSchema prevents serving requests against an unmigrated database.
func RequireCurrentSchema(ctx context.Context, pool *pgxpool.Pool) error {
	db := stdlib.OpenDBFromPool(pool)
	defer db.Close()
	provider, err := MigrationProvider(db)
	if err != nil {
		return err
	}
	current, target, err := provider.GetVersions(ctx)
	if err != nil {
		return err
	}
	if current != target {
		return fmt.Errorf("database schema version %d, server requires %d; run go run ./cmd/db up", current, target)
	}
	pending, err := provider.HasPending(ctx)
	if err != nil {
		return err
	}
	if pending {
		return errors.New("pending database migrations; run go run ./cmd/db up")
	}
	return nil
}

// MigrationProvider uses the same embedded SQL files consumed by sqlc. Goose
// records schema versions and serializes migrations with a PostgreSQL lock.
func MigrationProvider(db *sql.DB) (*goose.Provider, error) {
	files, err := fs.Sub(migrations, "migrations")
	if err != nil {
		return nil, err
	}
	locker, err := lock.NewPostgresSessionLocker()
	if err != nil {
		return nil, err
	}
	return goose.NewProvider(goose.DialectPostgres, db, files,
		goose.WithDisableGlobalRegistry(true), goose.WithSessionLocker(locker))
}
