package instrument

import (
	"context"
	"sort"

	"github.com/mgwinsor/meridian/backend/internal/currency"
)

type Repository interface {
	Save(ctx context.Context, instrument Instrument) error
	FindByID(ctx context.Context, id ID) (Instrument, error)
	List(ctx context.Context) ([]Instrument, error)
}

type Service struct{ repository Repository }

func NewService(repository Repository) Service { return Service{repository: repository} }

func (s Service) CreateInstrument(ctx context.Context, kind Kind, symbol, name string, quoteCurrency currency.Code) (Instrument, error) {
	instrument, err := New(NewID(), kind, symbol, name, quoteCurrency)
	if err != nil {
		return Instrument{}, err
	}
	if err := s.repository.Save(ctx, instrument); err != nil {
		return Instrument{}, err
	}
	return instrument, nil
}

func (s Service) GetByID(ctx context.Context, id ID) (Instrument, error) {
	return s.repository.FindByID(ctx, id)
}

func (s Service) ListInstruments(ctx context.Context) ([]Instrument, error) {
	instruments, err := s.repository.List(ctx)
	if err != nil {
		return nil, err
	}
	sort.Slice(instruments, func(i, j int) bool { return instruments[i].ID.String() < instruments[j].ID.String() })
	return instruments, nil
}
