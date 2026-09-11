package money

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/mgwinsor/meridian/backend/internal/currency"
)

var ErrInvalidAmount = errors.New("invalid amount")

type Amount struct {
	Currency   currency.Code
	minorUnits int64
}

func Parse(code currency.Code, value string) (Amount, error) {
	digits, err := code.MinorUnitDigits()
	if err != nil {
		return Amount{}, fmt.Errorf("%w: %v", ErrInvalidAmount, err)
	}

	value = strings.TrimSpace(value)
	if value == "" || strings.HasPrefix(value, "-") {
		return Amount{}, ErrInvalidAmount
	}

	parts := strings.Split(value, ".")
	if len(parts) > 2 || parts[0] == "" || !isDigits(parts[0]) {
		return Amount{}, ErrInvalidAmount
	}

	fraction := ""
	if len(parts) == 2 {
		fraction = parts[1]
		if fraction == "" || !isDigits(fraction) || len(fraction) > digits {
			return Amount{}, ErrInvalidAmount
		}
	}

	fraction += strings.Repeat("0", digits-len(fraction))
	minorUnitValue := strings.TrimLeft(parts[0]+fraction, "0")
	if minorUnitValue == "" {
		minorUnitValue = "0"
	}

	parsed, err := strconv.ParseInt(minorUnitValue, 10, 64)
	if err != nil {
		return Amount{}, ErrInvalidAmount
	}

	return Amount{Currency: code, minorUnits: parsed}, nil
}

func (a Amount) String() string {
	digits, err := a.Currency.MinorUnitDigits()
	if err != nil {
		return ""
	}

	value := strconv.FormatInt(a.minorUnits, 10)
	if digits == 0 {
		return value
	}

	if len(value) <= digits {
		value = strings.Repeat("0", digits-len(value)+1) + value
	}

	separator := len(value) - digits
	return value[:separator] + "." + value[separator:]
}

func isDigits(value string) bool {
	for _, char := range value {
		if char < '0' || char > '9' {
			return false
		}
	}

	return true
}
