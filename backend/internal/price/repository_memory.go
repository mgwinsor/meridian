package price

import (
	"context"
	"sort"
	"sync"
	"time"

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

func (r *MemoryRepository) Latest(_ context.Context, ids []instrument.ID, cutoff time.Time) ([]Observation, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := []Observation{}
	seen := map[instrument.ID]bool{}
	for _, id := range ids {
		if seen[id] {
			continue
		}
		seen[id] = true
		var latest *Observation
		for _, item := range r.observations[id] {
			if !item.ObservedAt.After(cutoff) && (latest == nil || !item.ObservedAt.Before(latest.ObservedAt)) {
				copy := item
				latest = &copy
			}
		}
		if latest != nil {
			result = append(result, *latest)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].InstrumentID.String() < result[j].InstrumentID.String() })
	return result, nil
}
