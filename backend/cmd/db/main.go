// db runs goose migrations. init also creates a missing application database.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/mgwinsor/meridian/backend/internal/database"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	if len(os.Args) != 2 {
		return fmt.Errorf("usage: db init|up|down|status")
	}
	command := os.Args[1]
	switch command {
	case "init", "up", "down", "status":
	default:
		return fmt.Errorf("unknown command %q: use init, up, down, or status", command)
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	url := database.URL()
	if command == "init" {
		if err := database.Create(ctx, url); err != nil {
			return fmt.Errorf("create database: %w", err)
		}
	}
	config, err := pgx.ParseConfig(url)
	if err != nil {
		return err
	}
	db := stdlib.OpenDB(*config)
	defer db.Close()
	provider, err := database.MigrationProvider(db)
	if err != nil {
		return err
	}
	switch command {
	case "init", "up":
		results, err := provider.Up(ctx)
		if err != nil {
			return err
		}
		for _, result := range results {
			fmt.Printf("applied %s\n", result.Source.Path)
		}
	case "down":
		result, err := provider.Down(ctx)
		if err != nil {
			return err
		}
		fmt.Printf("rolled back %s\n", result.Source.Path)
	case "status":
		statuses, err := provider.Status(ctx)
		if err != nil {
			return err
		}
		for _, status := range statuses {
			fmt.Printf("%s %s\n", status.State, status.Source.Path)
		}
	}
	return nil
}
