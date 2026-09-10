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
