package cash

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mgwinsor/meridian/backend/internal/account"
	"github.com/mgwinsor/meridian/backend/internal/cash/postgres"
	"github.com/mgwinsor/meridian/backend/internal/currency"
	"github.com/mgwinsor/meridian/backend/internal/money"
)

type PostgresRepository struct{ queries *postgres.Queries }

var _ Repository = (*PostgresRepository)(nil)

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{queries: postgres.New(pool)}
}

func (r *PostgresRepository) Save(ctx context.Context, balance Balance) error {
	return r.queries.Save(ctx, postgres.SaveParams{
		AccountID: uuid.MustParse(balance.AccountID.String()),
		Currency:  balance.Amount.Currency().String(), MinorUnits: balance.Amount.MinorUnits(),
	})
}

func (r *PostgresRepository) ListByAccount(ctx context.Context, accountID account.ID) ([]Balance, error) {
	rows, err := r.queries.ListByAccount(ctx, uuid.MustParse(accountID.String()))
	if err != nil {
		return nil, err
	}
	balances := make([]Balance, 0, len(rows))
	for _, row := range rows {
		code, err := currency.Parse(row.Currency)
		if err != nil {
			return nil, err
		}
		amount, err := money.FromMinorUnits(code, row.MinorUnits)
		if err != nil {
			return nil, err
		}
		balances = append(balances, Balance{AccountID: accountID, Amount: amount})
	}
	return balances, nil
}
