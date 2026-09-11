package account

import (
	"context"
	"errors"
	"testing"
)

type stubRepository struct {
	save       func(context.Context, Account) error
	findByID   func(context.Context, ID) (Account, error)
	list       func(context.Context) ([]Account, error)
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

func (r *stubRepository) List(ctx context.Context) ([]Account, error) {
	if r.list != nil {
		return r.list(ctx)
	}
	return []Account{}, nil
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
		list: func(context.Context) ([]Account, error) {
			return nil, wantErr
		},
	}
	service := NewService(repository)

	if _, err := service.CreateAccount(context.Background(), "Primary"); !errors.Is(err, wantErr) {
		t.Errorf("CreateAccount() error = %v, want %v", err, wantErr)
	}
	if _, err := service.GetByID(context.Background(), NewID()); !errors.Is(err, wantErr) {
		t.Errorf("GetByID() error = %v, want %v", err, wantErr)
	}
	if _, err := service.ListAccounts(context.Background()); !errors.Is(err, wantErr) {
		t.Errorf("ListAccounts() error = %v, want %v", err, wantErr)
	}
}

func TestServiceListAccountsSortsByCanonicalID(t *testing.T) {
	firstID, err := ParseID("00000000-0000-0000-0000-000000000001")
	if err != nil {
		t.Fatalf("ParseID() unexpected error: %v", err)
	}
	secondID, err := ParseID("00000000-0000-0000-0000-000000000002")
	if err != nil {
		t.Fatalf("ParseID() unexpected error: %v", err)
	}
	repository := &stubRepository{
		list: func(context.Context) ([]Account, error) {
			return []Account{
				{ID: secondID, Name: "Second"},
				{ID: firstID, Name: "First"},
			}, nil
		},
	}

	got, err := NewService(repository).ListAccounts(context.Background())
	if err != nil {
		t.Fatalf("ListAccounts() unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("ListAccounts() returned %d accounts, want 2", len(got))
	}
	if got[0].ID != firstID || got[1].ID != secondID {
		t.Errorf("ListAccounts() IDs = [%s, %s], want [%s, %s]", got[0].ID, got[1].ID, firstID, secondID)
	}
}
