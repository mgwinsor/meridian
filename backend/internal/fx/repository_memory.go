package fx

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/mgwinsor/meridian/backend/internal/currency"
)

type MemoryRepository struct {
	mu           sync.RWMutex
	observations []Observation
}

var _ Repository = (*MemoryRepository)(nil)

func NewMemoryRepository() *MemoryRepository { return &MemoryRepository{} }
func (r *MemoryRepository) Save(_ context.Context, item Observation) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.observations = append(r.observations, item)
	return nil
}
func (r *MemoryRepository) List(_ context.Context, code currency.Code) ([]Observation, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := []Observation{}
	for _, item := range r.observations {
		if item.Currency == code {
			result = append(result, item)
		}
	}
	sort.SliceStable(result, func(i, j int) bool { return result[i].ObservedAt.Before(result[j].ObservedAt) })
	return result, nil
}
func (r *MemoryRepository) Latest(_ context.Context, cutoff time.Time) ([]Observation, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	latest := map[currency.Code]Observation{}
	for _, item := range r.observations {
		previous, ok := latest[item.Currency]
		if !item.ObservedAt.After(cutoff) && (!ok || !item.ObservedAt.Before(previous.ObservedAt)) {
			latest[item.Currency] = item
		}
	}
	result := []Observation{}
	for _, item := range latest {
		result = append(result, item)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Currency.String() < result[j].Currency.String() })
	return result, nil
}
