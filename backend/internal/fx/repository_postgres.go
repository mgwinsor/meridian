package fx

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/mgwinsor/meridian/backend/internal/currency"
	"github.com/mgwinsor/meridian/backend/internal/fx/postgres"
)

type PostgresRepository struct{ queries *postgres.Queries }

var _ Repository = (*PostgresRepository)(nil)

func NewPostgresRepository(db postgres.DBTX) *PostgresRepository {
	return &PostgresRepository{postgres.New(db)}
}

func (r *PostgresRepository) Save(ctx context.Context, item Observation) error {
	return r.queries.Save(ctx, postgres.SaveParams{
		Currency:   item.Currency.String(),
		Rate:       item.Rate.value,
		ObservedAt: pgtype.Timestamptz{Time: item.ObservedAt, Valid: true},
	})
}

func fromRows(rows []postgres.FxObservation) ([]Observation, error) {
	result := make([]Observation, 0, len(rows))
	for _, row := range rows {
		code, err := currency.Parse(row.Currency)
		if err != nil {
			return nil, err
		}
		rate, err := ParseRate(row.Rate.String())
		if err != nil {
			return nil, err
		}
		item, err := New(code, rate, row.ObservedAt.Time)
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, nil
}

func (r *PostgresRepository) List(ctx context.Context, code currency.Code) ([]Observation, error) {
	rows, err := r.queries.List(ctx, code.String())
	if err != nil {
		return nil, err
	}
	return fromRows(rows)
}

func (r *PostgresRepository) Latest(ctx context.Context, cutoff time.Time) ([]Observation, error) {
	rows, err := r.queries.Latest(ctx, pgtype.Timestamptz{Time: cutoff, Valid: true})
	if err != nil {
		return nil, err
	}
	return fromRows(rows)
}
