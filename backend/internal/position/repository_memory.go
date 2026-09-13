package position

import (
	"context"
	"sync"

	"github.com/mgwinsor/meridian/backend/internal/account"
	"github.com/mgwinsor/meridian/backend/internal/instrument"
)

type MemoryRepository struct {
	mu        sync.RWMutex
	positions map[account.ID]map[instrument.ID]Position
}

var _ Repository = (*MemoryRepository)(nil)

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		positions: make(map[account.ID]map[instrument.ID]Position),
	}
}

func (r *MemoryRepository) Save(_ context.Context, position Position) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	accountPositions, ok := r.positions[position.AccountID]
	if !ok {
		accountPositions = make(map[instrument.ID]Position)
		r.positions[position.AccountID] = accountPositions
	}

	accountPositions[position.InstrumentID] = position

	return nil
}

func (r *MemoryRepository) ListByAccount(
	_ context.Context,
	accountID account.ID,
) ([]Position, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	accountPositions := r.positions[accountID]
	positions := make([]Position, 0, len(accountPositions))
	for _, position := range accountPositions {
		positions = append(positions, position)
	}

	return positions, nil
}
