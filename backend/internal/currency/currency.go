package currency

import (
	"errors"
	"strings"
)

var ErrUnsupported = errors.New("unsupported currency")

type Code struct {
	value string
}

func Parse(value string) (Code, error) {
	value = strings.ToUpper(strings.TrimSpace(value))

	switch value {
	case "USD", "SGD", "VND":
		return Code{value: value}, nil
	default:
		return Code{}, ErrUnsupported
	}
}

func (c Code) String() string {
	return c.value
}

func (c Code) MinorUnitDigits() (int, error) {
	switch c.value {
	case "USD", "SGD":
		return 2, nil
	case "VND":
		return 0, nil
	default:
		return 0, ErrUnsupported
	}
}
