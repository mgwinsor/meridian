package cash_test

import (
	"context"
	"sync"
	"testing"

	"github.com/mgwinsor/meridian/backend/internal/account"
	"github.com/mgwinsor/meridian/backend/internal/cash"
	"github.com/mgwinsor/meridian/backend/internal/currency"
	"github.com/mgwinsor/meridian/backend/internal/money"
)

func TestMemoryRepositorySavesListsAndReplacesBalances(t *testing.T) {
	repository := cash.NewMemoryRepository()
	primaryID := account.NewID()
	secondaryID := account.NewID()

	repositoryBalances := []cash.Balance{
		newBalance(t, primaryID, "USD", "10.00"),
		newBalance(t, primaryID, "SGD", "20.00"),
		newBalance(t, secondaryID, "USD", "99.00"),
		newBalance(t, primaryID, "USD", "15.00"),
	}
	for _, balance := range repositoryBalances {
		if err := repository.Save(context.Background(), balance); err != nil {
			t.Fatalf("Save() unexpected error: %v", err)
		}
	}

	got, err := repository.ListByAccount(context.Background(), primaryID)
	if err != nil {
		t.Fatalf("ListByAccount() unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("ListByAccount() returned %d balances, want 2", len(got))
	}

	amounts := make(map[string]string)
	for _, balance := range got {
		amounts[balance.Amount.Currency.String()] = balance.Amount.String()
	}
	if amounts["USD"] != "15.00" {
		t.Errorf("USD balance = %q, want %q", amounts["USD"], "15.00")
	}
	if amounts["SGD"] != "20.00" {
		t.Errorf("SGD balance = %q, want %q", amounts["SGD"], "20.00")
	}
}

func TestMemoryRepositoryReturnsEmptySlice(t *testing.T) {
	repository := cash.NewMemoryRepository()

	got, err := repository.ListByAccount(context.Background(), account.NewID())
	if err != nil {
		t.Fatalf("ListByAccount() unexpected error: %v", err)
	}
	if got == nil {
		t.Fatal("ListByAccount() returned nil, want empty slice")
	}
	if len(got) != 0 {
		t.Fatalf("ListByAccount() returned %d balances, want 0", len(got))
	}
}

func TestMemoryRepositoryRetainsZeroBalance(t *testing.T) {
	repository := cash.NewMemoryRepository()
	accountID := account.NewID()
	zero := newBalance(t, accountID, "USD", "0")

	if err := repository.Save(context.Background(), zero); err != nil {
		t.Fatalf("Save() unexpected error: %v", err)
	}

	got, err := repository.ListByAccount(context.Background(), accountID)
	if err != nil {
		t.Fatalf("ListByAccount() unexpected error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("ListByAccount() returned %d balances, want 1", len(got))
	}
	if got[0].Amount.String() != "0.00" {
		t.Errorf("zero balance = %q, want %q", got[0].Amount.String(), "0.00")
	}
}

func TestMemoryRepositorySupportsConcurrentAccess(t *testing.T) {
	repository := cash.NewMemoryRepository()
	accountID := account.NewID()
	balances := []cash.Balance{
		newBalance(t, accountID, "USD", "1.00"),
		newBalance(t, accountID, "SGD", "2.00"),
		newBalance(t, accountID, "VND", "3"),
	}

	var group sync.WaitGroup
	for _, balance := range balances {
		balance := balance
		group.Add(1)
		go func() {
			defer group.Done()
			if err := repository.Save(context.Background(), balance); err != nil {
				t.Errorf("Save() unexpected error: %v", err)
			}
			if _, err := repository.ListByAccount(context.Background(), accountID); err != nil {
				t.Errorf("ListByAccount() unexpected error: %v", err)
			}
		}()
	}
	group.Wait()

	got, err := repository.ListByAccount(context.Background(), accountID)
	if err != nil {
		t.Fatalf("ListByAccount() unexpected error: %v", err)
	}
	if len(got) != len(balances) {
		t.Fatalf("ListByAccount() returned %d balances, want %d", len(got), len(balances))
	}
}

func newBalance(t *testing.T, accountID account.ID, codeValue, amountValue string) cash.Balance {
	t.Helper()

	code, err := currency.Parse(codeValue)
	if err != nil {
		t.Fatalf("currency.Parse() unexpected error: %v", err)
	}
	amount, err := money.Parse(code, amountValue)
	if err != nil {
		t.Fatalf("money.Parse() unexpected error: %v", err)
	}

	return cash.Balance{AccountID: accountID, Amount: amount}
}
