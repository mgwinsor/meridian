package account

import (
	"errors"
	"strings"

	"github.com/google/uuid"
)

type ID struct {
	value uuid.UUID
}

func NewID() ID {
	return ID{value: uuid.New()}
}

func ParseID(value string) (ID, error) {
	id, err := uuid.Parse(value)
	if err != nil {
		return ID{}, err
	}

	return ID{value: id}, nil
}

func (id ID) String() string {
	return id.value.String()
}

type Account struct {
	ID   ID
	Name string
}

func New(id ID, name string) (Account, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Account{}, errors.New("account name cannot be empty")
	}

	return Account{ID: id, Name: name}, nil
}
