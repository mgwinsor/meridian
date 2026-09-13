package property

import (
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/mgwinsor/meridian/backend/internal/money"
)

var ErrInvalidName = errors.New("property name cannot be empty")
var ErrInvalidID = errors.New("invalid property ID")
var ErrNotFound = errors.New("property not found")

type ID struct {
	value uuid.UUID
}

func NewID() ID { return ID{value: uuid.New()} }

func ParseID(value string) (ID, error) {
	id, err := uuid.Parse(value)
	if err != nil || id.String() != value {
		return ID{}, ErrInvalidID
	}
	return ID{value: id}, nil
}

func (id ID) String() string { return id.value.String() }

type Property struct {
	ID    ID
	Name  string
	Value money.Amount
}

func New(id ID, name string, value money.Amount) (Property, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Property{}, ErrInvalidName
	}
	return Property{ID: id, Name: name, Value: value}, nil
}
