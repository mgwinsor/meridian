package cash

import (
	"context"
	"sort"

	"github.com/mgwinsor/meridian/backend/internal/account"
	"github.com/mgwinsor/meridian/backend/internal/money"
)

type AccountFinder interface {
	FindByID(ctx context.Context, id account.ID) (account.Account, error)
}

type Repository interface {
	Save(ctx context.Context, balance Balance) error
	ListByAccount(ctx context.Context, accountID account.ID) ([]Balance, error)
}

type Service struct {
	accounts   AccountFinder
	repository Repository
}

func NewService(accounts AccountFinder, repository Repository) Service {
	return Service{
		accounts:   accounts,
		repository: repository,
	}
}

func (s Service) SetBalance(ctx context.Context, accountID account.ID, amount money.Amount) (Balance, error) {
	if _, err := s.accounts.FindByID(ctx, accountID); err != nil {
		return Balance{}, err
	}

	balance := Balance{AccountID: accountID, Amount: amount}
	if err := s.repository.Save(ctx, balance); err != nil {
		return Balance{}, err
	}

	return balance, nil
}

func (s Service) ListBalances(ctx context.Context, accountID account.ID) ([]Balance, error) {
	if _, err := s.accounts.FindByID(ctx, accountID); err != nil {
		return nil, err
	}

	balances, err := s.repository.ListByAccount(ctx, accountID)
	if err != nil {
		return nil, err
	}

	sort.Slice(balances, func(i, j int) bool {
		return balances[i].Amount.Currency.String() < balances[j].Amount.Currency.String()
	})

	return balances, nil
}
