package position

import (
	"errors"
	"strings"

	"github.com/mgwinsor/meridian/backend/internal/account"
	"github.com/mgwinsor/meridian/backend/internal/instrument"
	"github.com/shopspring/decimal"
)

var ErrInvalidQuantity = errors.New("invalid quantity")

// Quantities fit PostgreSQL NUMERIC(38,18): 20 integer and 18 fractional digits.
const quantityIntegerDigits = 20
const quantityFractionDigits = 18

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
	whole := strings.TrimLeft(parts[0], "0")
	fraction := ""
	if len(parts) == 2 {
		fraction = strings.TrimRight(parts[1], "0")
	}
	if len(whole) > quantityIntegerDigits || len(fraction) > quantityFractionDigits {
		return Quantity{}, ErrInvalidQuantity
	}
	if whole == "" {
		whole = "0"
	}
	value = whole
	if fraction != "" {
		value += "." + fraction
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
