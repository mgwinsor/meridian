package fx

import (
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/mgwinsor/meridian/backend/internal/currency"
	"github.com/shopspring/decimal"
)

var ErrInvalidRate = errors.New("rate must be a positive decimal with at most 20 integer and 18 fractional digits")
var ErrInvalidObservedAt = errors.New("invalid observation timestamp")
var syntax = regexp.MustCompile(`^[0-9]+(\.[0-9]+)?$`)

type Rate struct{ value decimal.Decimal }

func ParseRate(value string) (Rate, error) {
	value = strings.TrimSpace(value)
	if !syntax.MatchString(value) {
		return Rate{}, ErrInvalidRate
	}
	parts := strings.Split(value, ".")
	if len(strings.TrimLeft(parts[0], "0")) > 20 || (len(parts) == 2 && len(strings.TrimRight(parts[1], "0")) > 18) {
		return Rate{}, ErrInvalidRate
	}
	parsed, err := decimal.NewFromString(value)
	if err != nil || !parsed.IsPositive() {
		return Rate{}, ErrInvalidRate
	}
	return Rate{parsed}, nil
}
func (r Rate) String() string { return r.value.String() }

type Observation struct {
	Currency   currency.Code
	Rate       Rate
	ObservedAt time.Time
}

func New(code currency.Code, rate Rate, observedAt time.Time) (Observation, error) {
	if code.String() != "SGD" && code.String() != "VND" {
		return Observation{}, currency.ErrUnsupported
	}
	if !rate.value.IsPositive() {
		return Observation{}, ErrInvalidRate
	}
	observedAt = observedAt.UTC().Truncate(time.Microsecond)
	if observedAt.IsZero() || observedAt.Year() < 0 || observedAt.Year() > 9999 {
		return Observation{}, ErrInvalidObservedAt
	}
	return Observation{code, rate, observedAt}, nil
}
