package instrument

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/mgwinsor/meridian/backend/internal/currency"
)

func TestServiceLifecycle(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryRepository()
	svc := NewService(repo)
	empty, err := svc.ListInstruments(ctx)
	if err != nil || empty == nil || len(empty) != 0 {
		t.Fatalf("empty list: %v, %v", empty, err)
	}
	usd, _ := currency.Parse("USD")
	a, err := svc.CreateInstrument(ctx, KindStock, " AAPL ", " Apple ", usd)
	if err != nil {
		t.Fatal(err)
	}
	b, err := svc.CreateInstrument(ctx, KindStock, "AAPL", "Apple", usd)
	if err != nil || a.ID == b.ID {
		t.Fatalf("duplicate metadata must have independent identity: %v", err)
	}
	got, err := svc.GetByID(ctx, a.ID)
	if err != nil || got != a {
		t.Fatalf("retrieve: %+v, %v", got, err)
	}
	list, err := svc.ListInstruments(ctx)
	if err != nil || len(list) != 2 || list[0].ID.String() >= list[1].ID.String() {
		t.Fatalf("ordered list: %+v, %v", list, err)
	}
	list[0].Name = "modified snapshot"
	got, err = svc.GetByID(ctx, list[0].ID)
	if err != nil || got.Name != "Apple" {
		t.Fatal("list leaked mutable storage")
	}
	if _, err := svc.GetByID(ctx, NewID()); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing: %v", err)
	}
	if _, err := svc.CreateInstrument(ctx, KindStock, "", "Apple", usd); !errors.Is(err, ErrInvalidSymbol) {
		t.Fatal(err)
	}
	list, _ = svc.ListInstruments(ctx)
	if len(list) != 2 {
		t.Fatal("invalid instrument persisted")
	}
}

type failingRepository struct{ err error }

func (r failingRepository) Save(context.Context, Instrument) error { return r.err }
func (r failingRepository) FindByID(context.Context, ID) (Instrument, error) {
	return Instrument{}, r.err
}
func (r failingRepository) List(context.Context) ([]Instrument, error) { return nil, r.err }

func TestServiceRepositoryErrors(t *testing.T) {
	want := errors.New("storage unavailable")
	svc := NewService(failingRepository{want})
	ctx := context.Background()
	usd, _ := currency.Parse("USD")
	_, createErr := svc.CreateInstrument(ctx, KindStock, "AAPL", "Apple", usd)
	_, getErr := svc.GetByID(ctx, NewID())
	_, listErr := svc.ListInstruments(ctx)
	for _, err := range []error{createErr, getErr, listErr} {
		if !errors.Is(err, want) {
			t.Fatalf("got %v, want %v", err, want)
		}
	}
}

func TestMemoryRepositoryConcurrentAccess(t *testing.T) {
	svc := NewService(NewMemoryRepository())
	ctx := context.Background()
	usd, _ := currency.Parse("USD")
	var wg sync.WaitGroup
	for range 40 {
		wg.Go(func() {
			created, err := svc.CreateInstrument(ctx, KindStock, "AAPL", "Apple", usd)
			if err != nil {
				t.Error(err)
				return
			}
			if got, err := svc.GetByID(ctx, created.ID); err != nil || got != created {
				t.Errorf("get: %+v, %v", got, err)
			}
			if _, err := svc.ListInstruments(ctx); err != nil {
				t.Error(err)
			}
		})
	}
	wg.Wait()
	list, err := svc.ListInstruments(ctx)
	if err != nil || len(list) != 40 {
		t.Fatalf("lost concurrent writes: %d, %v", len(list), err)
	}
}
