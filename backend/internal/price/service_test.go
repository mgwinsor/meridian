package price

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/mgwinsor/meridian/backend/internal/currency"
	"github.com/mgwinsor/meridian/backend/internal/instrument"
	"github.com/mgwinsor/meridian/backend/internal/money"
)

func TestServiceLifecycle(t *testing.T) {
	ctx := context.Background()
	instruments := instrument.NewMemoryRepository()
	id, otherID := instrument.NewID(), instrument.NewID()
	for _, id := range []instrument.ID{id, otherID} {
		if err := instruments.Save(ctx, instrument.Instrument{ID: id, QuoteCurrency: testAmount(t, "USD", "0").Currency()}); err != nil {
			t.Fatal(err)
		}
	}
	repository := NewMemoryRepository()
	service := NewService(instruments, repository)
	empty, err := service.ListObservations(ctx, id)
	if err != nil || empty == nil || len(empty) != 0 {
		t.Fatalf("empty list = %v, %v", empty, err)
	}
	at := time.Date(2026, 9, 14, 4, 0, 0, 0, time.UTC)
	later, err := service.RecordObservation(ctx, id, "12.34", at.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	earlier, err := service.RecordObservation(ctx, id, "0", at)
	if err != nil {
		t.Fatal(err)
	}
	// Equal instants retain insertion order after UTC normalization.
	sameInstant, err := service.RecordObservation(ctx, id, "16.00", at.In(time.FixedZone("SGT", 8*3600)))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.RecordObservation(ctx, id, earlier.Amount.String(), at); err != nil {
		t.Fatal(err)
	}
	got, err := service.ListObservations(ctx, id)
	want := []Observation{earlier, sameInstant, earlier, later}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("history = %+v, %v; want %+v", got, err, want)
	}
	got[0].Amount = later.Amount
	again, err := service.ListObservations(ctx, id)
	if err != nil || !reflect.DeepEqual(again, want) {
		t.Fatalf("snapshot mutated storage: %+v, %v", again, err)
	}
	empty, err = service.ListObservations(ctx, otherID)
	if err != nil || empty == nil || len(empty) != 0 {
		t.Fatalf("other instrument = %v, %v", empty, err)
	}
	missing := instrument.NewID()
	if _, err := service.RecordObservation(ctx, missing, earlier.Amount.String(), at); !errors.Is(err, instrument.ErrNotFound) {
		t.Fatalf("missing instrument record: %v", err)
	}
	if _, err := service.ListObservations(ctx, missing); !errors.Is(err, instrument.ErrNotFound) {
		t.Fatalf("missing instrument list: %v", err)
	}
	if stored, err := repository.ListByInstrument(ctx, missing); err != nil || len(stored) != 0 {
		t.Fatalf("missing instrument persisted: %v, %v", stored, err)
	}
	for _, tc := range []struct {
		amount string
		at     time.Time
		want   error
	}{
		{"", at, money.ErrInvalidAmount},
		{earlier.Amount.String(), time.Time{}, ErrInvalidObservedAt},
	} {
		if _, err := service.RecordObservation(ctx, id, tc.amount, tc.at); !errors.Is(err, tc.want) {
			t.Errorf("invalid observation: %v; want %v", err, tc.want)
		}
	}
	again, err = service.ListObservations(ctx, id)
	if err != nil || !reflect.DeepEqual(again, want) {
		t.Fatalf("failed writes changed history: %+v, %v", again, err)
	}
}

type stubInstrumentFinder struct{ err error }

func (f stubInstrumentFinder) FindByID(_ context.Context, id instrument.ID) (instrument.Instrument, error) {
	code, _ := currency.Parse("USD")
	return instrument.Instrument{ID: id, QuoteCurrency: code}, f.err
}

type failingRepository struct {
	err   error
	calls int
}

func (r *failingRepository) Save(context.Context, Observation) error {
	r.calls++
	return r.err
}

func (r *failingRepository) ListByInstrument(context.Context, instrument.ID) ([]Observation, error) {
	r.calls++
	return nil, r.err
}

func TestServiceDependencyErrors(t *testing.T) {
	want := errors.New("storage unavailable")
	for _, finderErr := range []error{nil, want} {
		repository := &failingRepository{err: want}
		service := NewService(stubInstrumentFinder{err: finderErr}, repository)
		ctx := context.Background()
		id := instrument.NewID()
		_, recordErr := service.RecordObservation(ctx, id, "1", time.Now())
		_, listErr := service.ListObservations(ctx, id)
		for _, err := range []error{recordErr, listErr} {
			if !errors.Is(err, want) {
				t.Fatalf("error = %v; want %v", err, want)
			}
		}
		if finderErr != nil && repository.calls != 0 {
			t.Fatal("repository called after instrument lookup failed")
		}
	}
}
