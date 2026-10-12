package price

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/mgwinsor/meridian/backend/internal/currency"
	"github.com/mgwinsor/meridian/backend/internal/instrument"
	"github.com/mgwinsor/meridian/backend/internal/money"
)

func TestMemoryRepositoryConcurrentAccess(t *testing.T) {
	repository := NewMemoryRepository()
	ctx := context.Background()
	id := instrument.NewID()
	observation, err := New(id, testAmount(t, "USD", "92233720368547758.07"), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for range 40 {
		wg.Go(func() {
			if err := repository.Save(ctx, observation); err != nil {
				t.Error(err)
			}
			list, err := repository.ListByInstrument(ctx, id)
			if err != nil {
				t.Error(err)
				return
			}
			for i := range list {
				if list[i] != observation {
					t.Errorf("observation = %+v", list[i])
				}
				list[i] = Observation{}
			}
		})
	}
	wg.Wait()
	list, err := repository.ListByInstrument(ctx, id)
	if err != nil || len(list) != 40 {
		t.Fatalf("lost concurrent observations: %d, %v", len(list), err)
	}
	for _, got := range list {
		if got != observation {
			t.Fatal("snapshot mutation leaked into storage")
		}
	}
}

func TestLatestCutoffAndTies(t *testing.T) {
	repo := NewMemoryRepository()
	ctx := context.Background()
	id := instrument.NewID()
	code, _ := currency.Parse("USD")
	at := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for _, item := range []struct {
		value string
		at    time.Time
	}{{"2", at}, {"9", at.Add(time.Hour)}, {"1", at.Add(-time.Hour)}, {"3", at}} {
		amount, _ := money.Parse(code, item.value)
		observation, _ := New(id, amount, item.at)
		if err := repo.Save(ctx, observation); err != nil {
			t.Fatal(err)
		}
	}
	rows, err := repo.Latest(ctx, []instrument.ID{id, instrument.NewID()}, at)
	if err != nil || len(rows) != 1 || rows[0].Amount.String() != "3.00" {
		t.Fatalf("%+v %v", rows, err)
	}
	rows, err = repo.Latest(ctx, []instrument.ID{id}, at.Add(-2*time.Hour))
	if err != nil || rows == nil || len(rows) != 0 {
		t.Fatalf("%+v %v", rows, err)
	}
}
