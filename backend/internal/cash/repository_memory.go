package cash

import (
	"context"
	"sync"

	"github.com/mgwinsor/meridian/backend/internal/account"
	"github.com/mgwinsor/meridian/backend/internal/currency"
)

type MemoryRepository struct {
	mu       sync.RWMutex
	balances map[account.ID]map[currency.Code]Balance
}

var _ Repository = (*MemoryRepository)(nil)

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		balances: make(map[account.ID]map[currency.Code]Balance),
	}
}

func (r *MemoryRepository) Save(_ context.Context, balance Balance) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	accountBalances, ok := r.balances[balance.AccountID]
	if !ok {
		accountBalances = make(map[currency.Code]Balance)
		r.balances[balance.AccountID] = accountBalances
	}

	accountBalances[balance.Amount.Currency] = balance

	return nil
}

func (r *MemoryRepository) ListByAccount(
	_ context.Context,
	accountID account.ID,
) ([]Balance, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	accountBalances := r.balances[accountID]
	balances := make([]Balance, 0, len(accountBalances))
	for _, balance := range accountBalances {
		balances = append(balances, balance)
	}

	return balances, nil
}
