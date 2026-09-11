package account

import (
	"context"
	"errors"
	"sort"
)

var ErrNotFound = errors.New("account not found")

type Repository interface {
	Save(ctx context.Context, account Account) error
	FindByID(ctx context.Context, id ID) (Account, error)
	List(ctx context.Context) ([]Account, error)
}

type Service struct {
	repository Repository
}

func NewService(repository Repository) Service {
	return Service{
		repository: repository,
	}
}

func (s Service) CreateAccount(ctx context.Context, name string) (Account, error) {
	account, err := New(NewID(), name)
	if err != nil {
		return Account{}, err
	}

	if err := s.repository.Save(ctx, account); err != nil {
		return Account{}, err
	}

	return account, nil
}

func (s Service) GetByID(ctx context.Context, id ID) (Account, error) {
	return s.repository.FindByID(ctx, id)
}

func (s Service) ListAccounts(ctx context.Context) ([]Account, error) {
	accounts, err := s.repository.List(ctx)
	if err != nil {
		return nil, err
	}

	sort.Slice(accounts, func(i, j int) bool {
		return accounts[i].ID.String() < accounts[j].ID.String()
	})

	return accounts, nil
}
