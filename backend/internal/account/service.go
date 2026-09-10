package account

import (
	"context"
	"errors"
)

var ErrNotFound = errors.New("account not found")

type Repository interface {
	Save(ctx context.Context, account Account) error
	FindByID(ctx context.Context, id ID) (Account, error)
}

type Service struct {
	repository Repository
}

func NewService(repository Repository) Service {
	return Service{
		repository: repository,
	}
}

func (s Service) Create(ctx context.Context, name string) (Account, error) {
	account, err := New(NewID(), name)
	if err != nil {
		return Account{}, err
	}

	if err := s.repository.Save(ctx, account); err != nil {
		return Account{}, err
	}

	return account, nil
}

func (s Service) Get(ctx context.Context, id ID) (Account, error) {
	return s.repository.FindByID(ctx, id)
}
