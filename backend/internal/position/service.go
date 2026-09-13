package position

import (
	"context"
	"sort"

	"github.com/mgwinsor/meridian/backend/internal/account"
	"github.com/mgwinsor/meridian/backend/internal/instrument"
)

type AccountFinder interface {
	FindByID(ctx context.Context, id account.ID) (account.Account, error)
}

type InstrumentFinder interface {
	FindByID(ctx context.Context, id instrument.ID) (instrument.Instrument, error)
}

type Repository interface {
	Save(ctx context.Context, position Position) error
	ListByAccount(ctx context.Context, accountID account.ID) ([]Position, error)
}

type Service struct {
	accounts    AccountFinder
	instruments InstrumentFinder
	repository  Repository
}

func NewService(accounts AccountFinder, instruments InstrumentFinder, repository Repository) Service {
	return Service{
		accounts:    accounts,
		instruments: instruments,
		repository:  repository,
	}
}

func (s Service) SetPosition(ctx context.Context, accountID account.ID, instrumentID instrument.ID, quantity Quantity) (Position, error) {
	if _, err := s.accounts.FindByID(ctx, accountID); err != nil {
		return Position{}, err
	}

	if _, err := s.instruments.FindByID(ctx, instrumentID); err != nil {
		return Position{}, err
	}

	position := Position{AccountID: accountID, InstrumentID: instrumentID, Quantity: quantity}
	if err := s.repository.Save(ctx, position); err != nil {
		return Position{}, err
	}

	return position, nil
}

func (s Service) ListPositions(ctx context.Context, accountID account.ID) ([]Position, error) {
	if _, err := s.accounts.FindByID(ctx, accountID); err != nil {
		return nil, err
	}

	positions, err := s.repository.ListByAccount(ctx, accountID)
	if err != nil {
		return nil, err
	}

	sort.Slice(positions, func(i, j int) bool {
		return positions[i].InstrumentID.String() < positions[j].InstrumentID.String()
	})

	return positions, nil
}
