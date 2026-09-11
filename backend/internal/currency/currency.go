package currency

import (
	"errors"
	"strings"
)

type Code struct {
	value string
}

func Parse(value string) (Code, error) {
	value = strings.ToUpper(strings.TrimSpace(value))

	switch value {
	case "USD", "SGD", "VND":
		return Code{value: value}, nil
	default:
		return Code{}, errors.New("unsupported currency")
	}
}

func (c Code) String() string {
	return c.value
}
