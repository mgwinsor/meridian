package price

import (
	"errors"
	"time"

	"github.com/mgwinsor/meridian/backend/internal/instrument"
	"github.com/mgwinsor/meridian/backend/internal/money"
)

var ErrInvalidObservedAt = errors.New("invalid observation timestamp")

type Observation struct {
	InstrumentID instrument.ID
	Amount       money.Amount
	ObservedAt   time.Time
}

func New(instrumentID instrument.ID, amount money.Amount, observedAt time.Time) (Observation, error) {
	if amount.String() == "" {
		return Observation{}, money.ErrInvalidAmount
	}
	observedAt = observedAt.UTC()
	if observedAt.IsZero() || observedAt.Year() < 0 || observedAt.Year() > 9999 {
		return Observation{}, ErrInvalidObservedAt
	}
	return Observation{InstrumentID: instrumentID, Amount: amount, ObservedAt: observedAt}, nil
}
