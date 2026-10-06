package property

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mgwinsor/meridian/backend/internal/currency"
	"github.com/mgwinsor/meridian/backend/internal/money"
	"github.com/mgwinsor/meridian/backend/internal/property/postgres"
)

type PostgresRepository struct{ queries *postgres.Queries }

var _ Repository = (*PostgresRepository)(nil)

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{queries: postgres.New(pool)}
}

func (r *PostgresRepository) Create(ctx context.Context, property Property) error {
	return r.queries.Create(ctx, postgres.CreateParams{
		ID: property.ID.value, Name: property.Name,
		Currency: property.Value.Currency().String(), MinorUnits: property.Value.MinorUnits(),
	})
}

func (r *PostgresRepository) List(ctx context.Context) ([]Property, error) {
	rows, err := r.queries.List(ctx)
	if err != nil {
		return nil, err
	}
	properties := make([]Property, 0, len(rows))
	for _, row := range rows {
		property, err := propertyFromRow(row)
		if err != nil {
			return nil, err
		}
		properties = append(properties, property)
	}
	return properties, nil
}

func (r *PostgresRepository) ReplaceValue(ctx context.Context, id ID, value money.Amount) (Property, error) {
	row, err := r.queries.ReplaceValue(ctx, postgres.ReplaceValueParams{
		ID: id.value, Currency: value.Currency().String(), MinorUnits: value.MinorUnits(),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return Property{}, ErrNotFound
	}
	if err != nil {
		return Property{}, err
	}
	return propertyFromRow(row)
}

func propertyFromRow(row postgres.Property) (Property, error) {
	code, err := currency.Parse(row.Currency)
	if err != nil {
		return Property{}, err
	}
	amount, err := money.FromMinorUnits(code, row.MinorUnits)
	if err != nil {
		return Property{}, err
	}
	return New(ID{value: row.ID}, row.Name, amount)
}
