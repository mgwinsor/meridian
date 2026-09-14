package price

import (
	"context"
	"sort"
	"time"

	"github.com/mgwinsor/meridian/backend/internal/instrument"
	"github.com/mgwinsor/meridian/backend/internal/money"
)

type InstrumentFinder interface {
	FindByID(ctx context.Context, id instrument.ID) (instrument.Instrument, error)
}

type Repository interface {
	Save(ctx context.Context, observation Observation) error
	ListByInstrument(ctx context.Context, instrumentID instrument.ID) ([]Observation, error)
}

type Service struct {
	instruments InstrumentFinder
	repository  Repository
}

func NewService(instruments InstrumentFinder, repository Repository) Service {
	return Service{instruments: instruments, repository: repository}
}

func (s Service) RecordObservation(ctx context.Context, instrumentID instrument.ID, value string, observedAt time.Time) (Observation, error) {
	instrument, err := s.instruments.FindByID(ctx, instrumentID)
	if err != nil {
		return Observation{}, err
	}
	amount, err := money.Parse(instrument.QuoteCurrency, value)
	if err != nil {
		return Observation{}, err
	}
	observation, err := New(instrumentID, amount, observedAt)
	if err != nil {
		return Observation{}, err
	}
	if err := s.repository.Save(ctx, observation); err != nil {
		return Observation{}, err
	}
	return observation, nil
}

func (s Service) ListObservations(ctx context.Context, instrumentID instrument.ID) ([]Observation, error) {
	if _, err := s.instruments.FindByID(ctx, instrumentID); err != nil {
		return nil, err
	}
	observations, err := s.repository.ListByInstrument(ctx, instrumentID)
	if err != nil {
		return nil, err
	}
	sort.SliceStable(observations, func(i, j int) bool {
		return observations[i].ObservedAt.Before(observations[j].ObservedAt)
	})
	return observations, nil
}
