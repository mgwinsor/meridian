package position

import (
	"context"
	"errors"
	"testing"

	"github.com/mgwinsor/meridian/backend/internal/account"
	"github.com/mgwinsor/meridian/backend/internal/instrument"
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
	save          func(context.Context, Position) error
	listByAccount func(context.Context, account.ID) ([]Position, error)
	saveCalled    bool
	listCalled    bool
}

func (r *stubRepository) Save(ctx context.Context, position Position) error {
	r.saveCalled = true
	if r.save != nil {
		return r.save(ctx, position)
	}
	return nil
}

func (r *stubRepository) ListByAccount(ctx context.Context, id account.ID) ([]Position, error) {
	r.listCalled = true
	if r.listByAccount != nil {
		return r.listByAccount(ctx, id)
	}
	return []Position{}, nil
}

func TestServiceSetPosition(t *testing.T) {
	accountID := account.NewID()
	repository := &stubRepository{}
	service := NewService(existingAccountFinder(), existingInstrumentFinder(), repository)
	quantity, _ := ParseQuantity("12.34")
	instrumentID := instrument.NewID()

	got, err := service.SetPosition(context.Background(), accountID, instrumentID, quantity)
	if err != nil {
		t.Fatalf("SetPosition() unexpected error: %v", err)
	}
	if got.AccountID != accountID || got.InstrumentID != instrumentID || !got.Quantity.Equal(quantity) {
		t.Errorf("SetPosition() = %+v, want supplied account, instrument, and quantity", got)
	}
	if !repository.saveCalled {
		t.Error("SetPosition() did not save the position")
	}
}

func TestServiceChecksAccountBeforeRepository(t *testing.T) {
	repository := &stubRepository{}
	service := NewService(stubAccountFinder{}, existingInstrumentFinder(), repository)

	_, err := service.SetPosition(
		context.Background(),
		account.NewID(),
		instrument.NewID(), Quantity{},
	)
	if !errors.Is(err, account.ErrNotFound) {
		t.Fatalf("SetPosition() error = %v, want account.ErrNotFound", err)
	}
	if repository.saveCalled {
		t.Error("SetPosition() saved a position for a missing account")
	}

	_, err = service.ListPositions(context.Background(), account.NewID())
	if !errors.Is(err, account.ErrNotFound) {
		t.Fatalf("ListPositions() error = %v, want account.ErrNotFound", err)
	}
	if repository.listCalled {
		t.Error("ListPositions() queried positions for a missing account")
	}
}

func TestServicePropagatesDependencyErrors(t *testing.T) {
	wantErr := errors.New("dependency unavailable")
	accountID := account.NewID()
	quantity := Quantity{}
	instrumentID := instrument.NewID()

	accountService := NewService(stubAccountFinder{
		findByID: func(context.Context, account.ID) (account.Account, error) {
			return account.Account{}, wantErr
		},
	}, existingInstrumentFinder(), &stubRepository{})
	if _, err := accountService.SetPosition(context.Background(), accountID, instrumentID, quantity); !errors.Is(err, wantErr) {
		t.Errorf("SetPosition() account error = %v, want %v", err, wantErr)
	}

	repository := &stubRepository{
		save: func(context.Context, Position) error { return wantErr },
		listByAccount: func(context.Context, account.ID) ([]Position, error) {
			return nil, wantErr
		},
	}
	service := NewService(existingAccountFinder(), existingInstrumentFinder(), repository)
	if _, err := service.SetPosition(context.Background(), accountID, instrumentID, quantity); !errors.Is(err, wantErr) {
		t.Errorf("SetPosition() repository error = %v, want %v", err, wantErr)
	}
	if _, err := service.ListPositions(context.Background(), accountID); !errors.Is(err, wantErr) {
		t.Errorf("ListPositions() repository error = %v, want %v", err, wantErr)
	}
}

func existingAccountFinder() stubAccountFinder {
	return stubAccountFinder{
		findByID: func(_ context.Context, id account.ID) (account.Account, error) {
			return account.Account{ID: id, Name: "Primary"}, nil
		},
	}
}

type stubInstrumentFinder struct{ err error }

func (f stubInstrumentFinder) FindByID(_ context.Context, id instrument.ID) (instrument.Instrument, error) {
	return instrument.Instrument{ID: id}, f.err
}
func existingInstrumentFinder() stubInstrumentFinder { return stubInstrumentFinder{} }

func TestServiceMissingInstrument(t *testing.T) {
	for _, want := range []error{instrument.ErrNotFound, errors.New("lookup failed")} {
		repository := &stubRepository{}
		service := NewService(existingAccountFinder(), stubInstrumentFinder{err: want}, repository)
		if _, err := service.SetPosition(context.Background(), account.NewID(), instrument.NewID(), Quantity{}); !errors.Is(err, want) {
			t.Fatalf("error = %v, want %v", err, want)
		}
		if repository.saveCalled {
			t.Fatal("saved after failed instrument lookup")
		}
	}
}
