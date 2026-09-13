package property

import (
	"context"
	"sort"

	"github.com/mgwinsor/meridian/backend/internal/money"
)

type Repository interface {
	Create(ctx context.Context, property Property) error
	List(ctx context.Context) ([]Property, error)
	ReplaceValue(ctx context.Context, id ID, value money.Amount) (Property, error)
}

type Service struct{ repository Repository }

func NewService(repository Repository) Service { return Service{repository: repository} }

func (s Service) CreateProperty(ctx context.Context, name string, value money.Amount) (Property, error) {
	property, err := New(NewID(), name, value)
	if err != nil {
		return Property{}, err
	}
	if err := s.repository.Create(ctx, property); err != nil {
		return Property{}, err
	}
	return property, nil
}

func (s Service) ListProperties(ctx context.Context) ([]Property, error) {
	properties, err := s.repository.List(ctx)
	if err != nil {
		return nil, err
	}
	sort.Slice(properties, func(i, j int) bool { return properties[i].ID.String() < properties[j].ID.String() })
	return properties, nil
}

func (s Service) SetValue(ctx context.Context, id ID, value money.Amount) (Property, error) {
	return s.repository.ReplaceValue(ctx, id, value)
}
