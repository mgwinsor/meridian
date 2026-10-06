package instrument

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mgwinsor/meridian/backend/internal/currency"
	"github.com/mgwinsor/meridian/backend/internal/instrument/postgres"
)

type PostgresRepository struct{ queries *postgres.Queries }

var _ Repository = (*PostgresRepository)(nil)

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{queries: postgres.New(pool)}
}

func (r *PostgresRepository) Save(ctx context.Context, instrument Instrument) error {
	return r.queries.Save(ctx, postgres.SaveParams{
		ID: instrument.ID.value, Kind: string(instrument.Kind), Symbol: instrument.Symbol,
		Name: instrument.Name, QuoteCurrency: instrument.QuoteCurrency.String(),
	})
}

func (r *PostgresRepository) FindByID(ctx context.Context, id ID) (Instrument, error) {
	row, err := r.queries.FindByID(ctx, id.value)
	if errors.Is(err, pgx.ErrNoRows) {
		return Instrument{}, ErrNotFound
	}
	if err != nil {
		return Instrument{}, err
	}
	return instrumentFromRow(row)
}

func (r *PostgresRepository) List(ctx context.Context) ([]Instrument, error) {
	rows, err := r.queries.List(ctx)
	if err != nil {
		return nil, err
	}
	instruments := make([]Instrument, 0, len(rows))
	for _, row := range rows {
		instrument, err := instrumentFromRow(row)
		if err != nil {
			return nil, err
		}
		instruments = append(instruments, instrument)
	}
	return instruments, nil
}

func instrumentFromRow(row postgres.Instrument) (Instrument, error) {
	code, err := currency.Parse(row.QuoteCurrency)
	if err != nil {
		return Instrument{}, err
	}
	return New(ID{value: row.ID}, Kind(row.Kind), row.Symbol, row.Name, code)
}
