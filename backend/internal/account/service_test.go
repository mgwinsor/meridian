package account

import (
	"context"
	"errors"
	"testing"
)

type stubRepository struct {
	save       func(context.Context, Account) error
	findByID   func(context.Context, ID) (Account, error)
	saveCalled bool
}

func (r *stubRepository) Save(ctx context.Context, account Account) error {
	r.saveCalled = true
	if r.save != nil {
		return r.save(ctx, account)
	}
	return nil
}

func (r *stubRepository) FindByID(ctx context.Context, id ID) (Account, error) {
	if r.findByID != nil {
		return r.findByID(ctx, id)
	}
	return Account{}, ErrNotFound
}

func TestServiceCreateAccount(t *testing.T) {
	repository := &stubRepository{}
	service := NewService(repository)

	created, err := service.CreateAccount(context.Background(), "  Primary  ")
	if err != nil {
		t.Fatalf("CreateAccount() unexpected error: %v", err)
	}
	if created.Name != "Primary" {
		t.Errorf("CreateAccount() name = %q, want %q", created.Name, "Primary")
	}
	if !repository.saveCalled {
		t.Error("CreateAccount() did not save the account")
	}
}

func TestServiceCreateAccountValidationDoesNotSave(t *testing.T) {
	repository := &stubRepository{}
	service := NewService(repository)

	_, err := service.CreateAccount(context.Background(), " ")
	if !errors.Is(err, ErrInvalidName) {
		t.Fatalf("CreateAccount() error = %v, want ErrInvalidName", err)
	}
	if repository.saveCalled {
		t.Error("CreateAccount() saved an invalid account")
	}
}

func TestServicePropagatesRepositoryErrors(t *testing.T) {
	wantErr := errors.New("repository unavailable")
	repository := &stubRepository{
		save: func(context.Context, Account) error { return wantErr },
		findByID: func(context.Context, ID) (Account, error) {
			return Account{}, wantErr
		},
	}
	service := NewService(repository)

	if _, err := service.CreateAccount(context.Background(), "Primary"); !errors.Is(err, wantErr) {
		t.Errorf("CreateAccount() error = %v, want %v", err, wantErr)
	}
	if _, err := service.GetByID(context.Background(), NewID()); !errors.Is(err, wantErr) {
		t.Errorf("GetByID() error = %v, want %v", err, wantErr)
	}
}
