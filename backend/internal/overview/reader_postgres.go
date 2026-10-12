package overview

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mgwinsor/meridian/backend/internal/account"
	"github.com/mgwinsor/meridian/backend/internal/cash"
	"github.com/mgwinsor/meridian/backend/internal/fx"
	"github.com/mgwinsor/meridian/backend/internal/instrument"
	"github.com/mgwinsor/meridian/backend/internal/position"
	"github.com/mgwinsor/meridian/backend/internal/price"
	"github.com/mgwinsor/meridian/backend/internal/property"
)

type PostgresReader struct{ pool *pgxpool.Pool }

func NewPostgresReader(pool *pgxpool.Pool) *PostgresReader { return &PostgresReader{pool} }
func (r *PostgresReader) Read(ctx context.Context, cutoff time.Time) (Inputs, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return Inputs{}, err
	}
	defer tx.Rollback(ctx)
	inputs, err := readInputs(ctx, tx, cutoff)
	if err != nil {
		return Inputs{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Inputs{}, err
	}
	return inputs, nil
}
func readInputs(ctx context.Context, tx pgx.Tx, cutoff time.Time) (Inputs, error) {
	var input Inputs
	var err error
	input.Accounts, err = account.NewPostgresRepository(tx).List(ctx)
	if err != nil {
		return Inputs{}, err
	}
	input.Instruments, err = instrument.NewPostgresRepository(tx).List(ctx)
	if err != nil {
		return Inputs{}, err
	}
	input.Cash, err = cash.NewPostgresRepository(tx).List(ctx)
	if err != nil {
		return Inputs{}, err
	}
	input.Positions, err = position.NewPostgresRepository(tx).List(ctx)
	if err != nil {
		return Inputs{}, err
	}
	input.Properties, err = property.NewPostgresRepository(tx).List(ctx)
	if err != nil {
		return Inputs{}, err
	}
	ids := []instrument.ID{}
	seen := map[instrument.ID]bool{}
	for _, p := range input.Positions {
		if !seen[p.InstrumentID] {
			ids = append(ids, p.InstrumentID)
			seen[p.InstrumentID] = true
		}
	}
	input.Prices, err = price.NewPostgresRepository(tx).Latest(ctx, ids, cutoff)
	if err != nil {
		return Inputs{}, err
	}
	input.Rates, err = fx.NewPostgresRepository(tx).Latest(ctx, cutoff)
	if err != nil {
		return Inputs{}, err
	}
	return input, nil
}
