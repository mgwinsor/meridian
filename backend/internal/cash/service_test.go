package cash

import (
	"context"
	"errors"
	"testing"

	"github.com/mgwinsor/meridian/backend/internal/account"
	"github.com/mgwinsor/meridian/backend/internal/currency"
	"github.com/mgwinsor/meridian/backend/internal/money"
)

type stubAccountFinder struct {
	findByID func(context.Context, account.ID) (account.Account, error)
}

func (f stubAccountFinder) FindByID(ctx context.Context, id account.ID) (account.Account, error) {
	if f.findByID != nil {
		return f.findByID(ctx, id)
	}
	return account.Account{}, account.ErrNotFound
}

type stubRepository struct {
	save          func(context.Context, Balance) error
	listByAccount func(context.Context, account.ID) ([]Balance, error)
	saveCalled    bool
	listCalled    bool
}

func (r *stubRepository) Save(ctx context.Context, balance Balance) error {
	r.saveCalled = true
	if r.save != nil {
		return r.save(ctx, balance)
	}
	return nil
}

func (r *stubRepository) ListByAccount(ctx context.Context, id account.ID) ([]Balance, error) {
	r.listCalled = true
	if r.listByAccount != nil {
		return r.listByAccount(ctx, id)
	}
	return []Balance{}, nil
}

func TestServiceSetBalance(t *testing.T) {
	accountID := account.NewID()
	repository := &stubRepository{}
	service := NewService(existingAccountFinder(), repository)
	amount := parseAmount(t, "USD", "12.34")

	got, err := service.SetBalance(context.Background(), accountID, amount)
	if err != nil {
		t.Fatalf("SetBalance() unexpected error: %v", err)
	}
	if got.AccountID != accountID || got.Amount != amount {
		t.Errorf("SetBalance() = %+v, want supplied account and amount", got)
	}
	if !repository.saveCalled {
		t.Error("SetBalance() did not save the balance")
	}
}

func TestServiceChecksAccountBeforeRepository(t *testing.T) {
	repository := &stubRepository{}
	service := NewService(stubAccountFinder{}, repository)

	_, err := service.SetBalance(
		context.Background(),
		account.NewID(),
		parseAmount(t, "USD", "1.00"),
	)
	if !errors.Is(err, account.ErrNotFound) {
		t.Fatalf("SetBalance() error = %v, want account.ErrNotFound", err)
	}
	if repository.saveCalled {
		t.Error("SetBalance() saved a balance for a missing account")
	}

	_, err = service.ListBalances(context.Background(), account.NewID())
	if !errors.Is(err, account.ErrNotFound) {
		t.Fatalf("ListBalances() error = %v, want account.ErrNotFound", err)
	}
	if repository.listCalled {
		t.Error("ListBalances() queried balances for a missing account")
	}
}

func TestServiceListsBalancesInCurrencyOrder(t *testing.T) {
	accountID := account.NewID()
	repository := &stubRepository{
		listByAccount: func(context.Context, account.ID) ([]Balance, error) {
			return []Balance{
				{AccountID: accountID, Amount: parseAmount(t, "VND", "3")},
				{AccountID: accountID, Amount: parseAmount(t, "SGD", "2.00")},
				{AccountID: accountID, Amount: parseAmount(t, "USD", "1.00")},
			}, nil
		},
	}
	service := NewService(existingAccountFinder(), repository)

	got, err := service.ListBalances(context.Background(), accountID)
	if err != nil {
		t.Fatalf("ListBalances() unexpected error: %v", err)
	}
	want := []string{"SGD", "USD", "VND"}
	for i, balance := range got {
		if balance.Amount.Currency.String() != want[i] {
			t.Errorf(
				"ListBalances()[%d] currency = %q, want %q",
				i,
				balance.Amount.Currency.String(),
				want[i],
			)
		}
	}
}

func TestServicePropagatesDependencyErrors(t *testing.T) {
	wantErr := errors.New("dependency unavailable")
	accountID := account.NewID()
	amount := parseAmount(t, "USD", "1.00")

	accountService := NewService(stubAccountFinder{
		findByID: func(context.Context, account.ID) (account.Account, error) {
			return account.Account{}, wantErr
		},
	}, &stubRepository{})
	if _, err := accountService.SetBalance(context.Background(), accountID, amount); !errors.Is(err, wantErr) {
		t.Errorf("SetBalance() account error = %v, want %v", err, wantErr)
	}

	repository := &stubRepository{
		save: func(context.Context, Balance) error { return wantErr },
		listByAccount: func(context.Context, account.ID) ([]Balance, error) {
			return nil, wantErr
		},
	}
	service := NewService(existingAccountFinder(), repository)
	if _, err := service.SetBalance(context.Background(), accountID, amount); !errors.Is(err, wantErr) {
		t.Errorf("SetBalance() repository error = %v, want %v", err, wantErr)
	}
	if _, err := service.ListBalances(context.Background(), accountID); !errors.Is(err, wantErr) {
		t.Errorf("ListBalances() repository error = %v, want %v", err, wantErr)
	}
}

func existingAccountFinder() stubAccountFinder {
	return stubAccountFinder{
		findByID: func(_ context.Context, id account.ID) (account.Account, error) {
			return account.Account{ID: id, Name: "Primary"}, nil
		},
	}
}

func parseAmount(t *testing.T, codeValue, amountValue string) money.Amount {
	t.Helper()

	code, err := currency.Parse(codeValue)
	if err != nil {
		t.Fatalf("currency.Parse() unexpected error: %v", err)
	}
	amount, err := money.Parse(code, amountValue)
	if err != nil {
		t.Fatalf("money.Parse() unexpected error: %v", err)
	}
	return amount
}
