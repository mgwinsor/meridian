package account

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mgwinsor/meridian/backend/internal/account/postgres"
)

type PostgresRepository struct{ queries *postgres.Queries }

var _ Repository = (*PostgresRepository)(nil)

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{queries: postgres.New(pool)}
}

func (r *PostgresRepository) Save(ctx context.Context, account Account) error {
	return r.queries.Save(ctx, postgres.SaveParams{ID: account.ID.value, Name: account.Name})
}

func (r *PostgresRepository) FindByID(ctx context.Context, id ID) (Account, error) {
	row, err := r.queries.FindByID(ctx, id.value)
	if errors.Is(err, pgx.ErrNoRows) {
		return Account{}, ErrNotFound
	}
	if err != nil {
		return Account{}, err
	}
	return New(ID{value: row.ID}, row.Name)
}

func (r *PostgresRepository) List(ctx context.Context) ([]Account, error) {
	rows, err := r.queries.List(ctx)
	if err != nil {
		return nil, err
	}
	accounts := make([]Account, 0, len(rows))
	for _, row := range rows {
		account, err := New(ID{value: row.ID}, row.Name)
		if err != nil {
			return nil, err
		}
		accounts = append(accounts, account)
	}
	return accounts, nil
}
