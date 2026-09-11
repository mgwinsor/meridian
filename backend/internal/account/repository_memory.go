package account

import (
	"context"
	"sync"
)

type MemoryRepository struct {
	mu       sync.RWMutex
	accounts map[ID]Account
}

var _ Repository = (*MemoryRepository)(nil)

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

func (r *MemoryRepository) FindByID(_ context.Context, id ID) (Account, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	account, ok := r.accounts[id]
	if !ok {
		return Account{}, ErrNotFound
	}

	return account, nil
}

func (r *MemoryRepository) List(_ context.Context) ([]Account, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	accounts := make([]Account, 0, len(r.accounts))
	for _, account := range r.accounts {
		accounts = append(accounts, account)
	}

	return accounts, nil
}
