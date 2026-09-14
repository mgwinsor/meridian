package price

import (
	"errors"
	"testing"
	"time"

	"github.com/mgwinsor/meridian/backend/internal/currency"
	"github.com/mgwinsor/meridian/backend/internal/instrument"
	"github.com/mgwinsor/meridian/backend/internal/money"
)

func testAmount(t *testing.T, code, value string) money.Amount {
	t.Helper()
	c, err := currency.Parse(code)
	if err != nil {
		t.Fatal(err)
	}
	amount, err := money.Parse(c, value)
	if err != nil {
		t.Fatal(err)
	}
	return amount
}

func TestNew(t *testing.T) {
	id := instrument.NewID()
	at := time.Date(2026, 9, 14, 12, 30, 45, 123456789, time.FixedZone("SGT", 8*60*60))
	for _, tc := range []struct{ currency, value string }{
		{"USD", "0.00"}, {"SGD", "92233720368547758.07"}, {"VND", "9223372036854775807"},
	} {
		t.Run(tc.currency, func(t *testing.T) {
			amount := testAmount(t, tc.currency, tc.value)
			got, err := New(id, amount, at)
			if err != nil || got.InstrumentID != id || got.Amount != amount || !got.ObservedAt.Equal(at) {
				t.Fatalf("New = %+v, %v", got, err)
			}
			if got.ObservedAt.Location() != time.UTC || got.ObservedAt.Format(time.RFC3339Nano) != "2026-09-14T04:30:45.123456789Z" {
				t.Fatalf("timestamp = %v", got.ObservedAt)
			}
		})
	}
	if _, err := New(id, money.Amount{}, at); !errors.Is(err, money.ErrInvalidAmount) {
		t.Fatalf("zero-value amount: %v", err)
	}
	for _, at := range []time.Time{
		{},
		time.Date(-1, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(10000, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(9999, 12, 31, 23, 0, 0, 0, time.FixedZone("west", -3600)),
	} {
		if _, err := New(id, testAmount(t, "USD", "1"), at); !errors.Is(err, ErrInvalidObservedAt) {
			t.Errorf("timestamp %v: %v", at, err)
		}
	}
}
