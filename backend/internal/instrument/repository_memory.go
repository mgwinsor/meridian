package instrument

import (
	"context"
	"sync"

	"github.com/mgwinsor/meridian/backend/internal/assetclass"
)

type MemoryRepository struct {
	mu          sync.RWMutex
	instruments map[ID]Instrument
}

var _ Repository = (*MemoryRepository)(nil)

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{instruments: make(map[ID]Instrument)}
}

func (r *MemoryRepository) Save(_ context.Context, instrument Instrument) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.instruments[instrument.ID] = instrument
	return nil
}

func (r *MemoryRepository) FindByID(_ context.Context, id ID) (Instrument, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	instrument, ok := r.instruments[id]
	if !ok {
		return Instrument{}, ErrNotFound
	}
	return instrument, nil
}

func (r *MemoryRepository) List(_ context.Context) ([]Instrument, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	instruments := make([]Instrument, 0, len(r.instruments))
	for _, instrument := range r.instruments {
		instruments = append(instruments, instrument)
	}
	return instruments, nil
}

func (r *MemoryRepository) ReplaceAssetClass(_ context.Context, id ID, class assetclass.Class) (Instrument, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	item, ok := r.instruments[id]
	if !ok {
		return Instrument{}, ErrNotFound
	}
	item.AssetClass = class
	r.instruments[id] = item
	return item, nil
}
