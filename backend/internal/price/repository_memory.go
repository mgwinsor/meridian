package price

import (
	"context"
	"sync"

	"github.com/mgwinsor/meridian/backend/internal/instrument"
)

type MemoryRepository struct {
	mu           sync.RWMutex
	observations map[instrument.ID][]Observation
}

var _ Repository = (*MemoryRepository)(nil)

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{observations: make(map[instrument.ID][]Observation)}
}

func (r *MemoryRepository) Save(_ context.Context, observation Observation) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.observations[observation.InstrumentID] = append(r.observations[observation.InstrumentID], observation)
	return nil
}

func (r *MemoryRepository) ListByInstrument(_ context.Context, instrumentID instrument.ID) ([]Observation, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	stored := r.observations[instrumentID]
	observations := make([]Observation, len(stored))
	copy(observations, stored)
	return observations, nil
}
