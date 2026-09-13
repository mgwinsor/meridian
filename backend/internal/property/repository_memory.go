package property

import (
	"context"
	"sync"

	"github.com/mgwinsor/meridian/backend/internal/money"
)

type MemoryRepository struct {
	mu         sync.RWMutex
	properties map[ID]Property
}

var _ Repository = (*MemoryRepository)(nil)

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{properties: make(map[ID]Property)}
}

func (r *MemoryRepository) Create(_ context.Context, property Property) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.properties[property.ID] = property
	return nil
}

func (r *MemoryRepository) List(_ context.Context) ([]Property, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	properties := make([]Property, 0, len(r.properties))
	for _, property := range r.properties {
		properties = append(properties, property)
	}
	return properties, nil
}

func (r *MemoryRepository) ReplaceValue(_ context.Context, id ID, value money.Amount) (Property, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	property, ok := r.properties[id]
	if !ok {
		return Property{}, ErrNotFound
	}
	property.Value = value
	r.properties[id] = property
	return property, nil
}
