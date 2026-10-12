package fx

import (
	"context"
	"time"

	"github.com/mgwinsor/meridian/backend/internal/currency"
)

type Repository interface {
	Save(context.Context, Observation) error
	List(context.Context, currency.Code) ([]Observation, error)
	Latest(context.Context, time.Time) ([]Observation, error)
}
type Service struct{ repository Repository }

func NewService(repository Repository) Service { return Service{repository} }

func (s Service) RecordObservation(
	ctx context.Context, code currency.Code, value string, at time.Time,
) (Observation, error) {
	rate, err := ParseRate(value)
	if err != nil {
		return Observation{}, err
	}
	item, err := New(code, rate, at)
	if err != nil {
		return Observation{}, err
	}
	if err := s.repository.Save(ctx, item); err != nil {
		return Observation{}, err
	}
	return item, nil
}

func (s Service) ListObservations(ctx context.Context, code currency.Code) ([]Observation, error) {
	if code.String() != "SGD" && code.String() != "VND" {
		return nil, currency.ErrUnsupported
	}
	return s.repository.List(ctx, code)
}
