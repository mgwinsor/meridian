package price

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/mgwinsor/meridian/backend/internal/instrument"
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
