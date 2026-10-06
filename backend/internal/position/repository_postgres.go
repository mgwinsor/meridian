package position

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mgwinsor/meridian/backend/internal/account"
	"github.com/mgwinsor/meridian/backend/internal/instrument"
	"github.com/mgwinsor/meridian/backend/internal/position/postgres"
)

type PostgresRepository struct{ queries *postgres.Queries }

var _ Repository = (*PostgresRepository)(nil)

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{queries: postgres.New(pool)}
}

func (r *PostgresRepository) Save(ctx context.Context, position Position) error {
	return r.queries.Save(ctx, postgres.SaveParams{
		AccountID:    uuid.MustParse(position.AccountID.String()),
		InstrumentID: uuid.MustParse(position.InstrumentID.String()), Quantity: position.Quantity.value,
	})
}

func (r *PostgresRepository) ListByAccount(ctx context.Context, accountID account.ID) ([]Position, error) {
	rows, err := r.queries.ListByAccount(ctx, uuid.MustParse(accountID.String()))
	if err != nil {
		return nil, err
	}
	positions := make([]Position, 0, len(rows))
	for _, row := range rows {
		instrumentID, err := instrument.ParseID(row.InstrumentID.String())
		if err != nil {
			return nil, err
		}
		quantity, err := ParseQuantity(row.Quantity.String())
		if err != nil {
			return nil, err
		}
		positions = append(positions, Position{AccountID: accountID, InstrumentID: instrumentID, Quantity: quantity})
	}
	return positions, nil
}
