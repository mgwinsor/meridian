package price

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/mgwinsor/meridian/backend/internal/currency"
	"github.com/mgwinsor/meridian/backend/internal/instrument"
	"github.com/mgwinsor/meridian/backend/internal/money"
	"github.com/mgwinsor/meridian/backend/internal/price/postgres"
)

type PostgresRepository struct{ queries *postgres.Queries }

var _ Repository = (*PostgresRepository)(nil)

func NewPostgresRepository(db postgres.DBTX) *PostgresRepository {
	return &PostgresRepository{queries: postgres.New(db)}
}

func (r *PostgresRepository) Save(ctx context.Context, observation Observation) error {
	return r.queries.Save(ctx, postgres.SaveParams{
		InstrumentID: uuid.MustParse(observation.InstrumentID.String()),
		Currency:     observation.Amount.Currency().String(), MinorUnits: observation.Amount.MinorUnits(),
		ObservedAt: pgtype.Timestamptz{Time: observation.ObservedAt, Valid: true},
	})
}

func (r *PostgresRepository) ListByInstrument(ctx context.Context, instrumentID instrument.ID) ([]Observation, error) {
	rows, err := r.queries.ListByInstrument(ctx, uuid.MustParse(instrumentID.String()))
	if err != nil {
		return nil, err
	}
	return fromRows(rows)
}

func fromRows(rows []postgres.PriceObservation) ([]Observation, error) {
	observations := make([]Observation, 0, len(rows))
	for _, row := range rows {
		instrumentID, err := instrument.ParseID(row.InstrumentID.String())
		if err != nil {
			return nil, err
		}
		code, err := currency.Parse(row.Currency)
		if err != nil {
			return nil, err
		}
		amount, err := money.FromMinorUnits(code, row.MinorUnits)
		if err != nil {
			return nil, err
		}
		observation, err := New(instrumentID, amount, row.ObservedAt.Time)
		if err != nil {
			return nil, err
		}
		observations = append(observations, observation)
	}
	return observations, nil
}

func (r *PostgresRepository) Latest(ctx context.Context, ids []instrument.ID, cutoff time.Time) ([]Observation, error) {
	values := make([]uuid.UUID, len(ids))
	for i, id := range ids {
		values[i] = uuid.MustParse(id.String())
	}
	rows, err := r.queries.Latest(ctx, postgres.LatestParams{Column1: values, ObservedAt: pgtype.Timestamptz{Time: cutoff, Valid: true}})
	if err != nil {
		return nil, err
	}
	return fromRows(rows)
}
