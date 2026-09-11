package account_test

import (
	"context"
	"errors"
	"testing"

	"github.com/mgwinsor/meridian/backend/internal/account"
)

func TestMemoryRepositorySaveAndFindByID(t *testing.T) {
	repository := account.NewMemoryRepository()
	want, err := account.New(account.NewID(), "Primary")
	if err != nil {
		t.Fatalf("New() unexpected error: %v", err)
	}

	if err := repository.Save(context.Background(), want); err != nil {
		t.Fatalf("Save() unexpected error: %v", err)
	}

	got, err := repository.FindByID(context.Background(), want.ID)
	if err != nil {
		t.Fatalf("FindByID() unexpected error: %v", err)
	}
	if got != want {
		t.Errorf("FindByID() = %#v, want %#v", got, want)
	}
}

func TestMemoryRepositoryFindByIDNotFound(t *testing.T) {
	repository := account.NewMemoryRepository()

	_, err := repository.FindByID(context.Background(), account.NewID())
	if !errors.Is(err, account.ErrNotFound) {
		t.Fatalf("FindByID() error = %v, want ErrNotFound", err)
	}
}

func TestMemoryRepositoryList(t *testing.T) {
	repository := account.NewMemoryRepository()
	first, err := account.New(account.NewID(), "Primary")
	if err != nil {
		t.Fatalf("New() unexpected error: %v", err)
	}
	second, err := account.New(account.NewID(), "Savings")
	if err != nil {
		t.Fatalf("New() unexpected error: %v", err)
	}

	for _, item := range []account.Account{first, second} {
		if err := repository.Save(context.Background(), item); err != nil {
			t.Fatalf("Save() unexpected error: %v", err)
		}
	}

	got, err := repository.List(context.Background())
	if err != nil {
		t.Fatalf("List() unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("List() returned %d accounts, want 2", len(got))
	}

	gotByID := make(map[string]account.Account, len(got))
	for _, item := range got {
		gotByID[item.ID.String()] = item
	}
	for _, want := range []account.Account{first, second} {
		if gotByID[want.ID.String()] != want {
			t.Errorf("List() missing account %#v", want)
		}
	}
}

func TestMemoryRepositoryListEmpty(t *testing.T) {
	repository := account.NewMemoryRepository()

	got, err := repository.List(context.Background())
	if err != nil {
		t.Fatalf("List() unexpected error: %v", err)
	}
	if got == nil {
		t.Fatal("List() returned nil, want empty slice")
	}
	if len(got) != 0 {
		t.Fatalf("List() returned %d accounts, want 0", len(got))
	}
}
