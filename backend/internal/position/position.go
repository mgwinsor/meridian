package position

import (
	"errors"
	"strings"

	"github.com/mgwinsor/meridian/backend/internal/account"
	"github.com/mgwinsor/meridian/backend/internal/instrument"
	"github.com/shopspring/decimal"
)

var ErrInvalidQuantity = errors.New("invalid quantity")

type Quantity struct{ value decimal.Decimal }

func ParseQuantity(value string) (Quantity, error) {
	value = strings.TrimSpace(value)
	parts := strings.Split(value, ".")
	if len(parts) > 2 {
		return Quantity{}, ErrInvalidQuantity
	}
	for _, part := range parts {
		if part == "" {
			return Quantity{}, ErrInvalidQuantity
		}
		for _, digit := range part {
			if digit < '0' || digit > '9' {
				return Quantity{}, ErrInvalidQuantity
			}
		}
	}
	parsed, err := decimal.NewFromString(value)
	if err != nil || parsed.IsNegative() {
		return Quantity{}, ErrInvalidQuantity
	}

	return Quantity{value: parsed}, nil
}

func (q Quantity) String() string { return q.value.String() }

func (q Quantity) Equal(other Quantity) bool { return q.value.Equal(other.value) }

type Position struct {
	AccountID    account.ID
	InstrumentID instrument.ID
	Quantity     Quantity
}
