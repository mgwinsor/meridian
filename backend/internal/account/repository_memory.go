package account

import (
	"context"
	"sync"
)

type MemoryRepository struct {
	mu       sync.RWMutex
	accounts map[ID]Account
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		accounts: make(map[ID]Account),
	}
}

func (r *MemoryRepository) Save(_ context.Context, account Account) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.accounts[account.ID] = account

	return nil
}

func (r *MemoryRepository) FindByID(_, id ID) (Account, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	account, ok := r.accounts[id]
	if !ok {
		return Account{}, ErrNotFound
	}

	return account, nil
}
