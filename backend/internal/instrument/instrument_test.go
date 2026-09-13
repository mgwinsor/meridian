package instrument

import (
	"errors"
	"strings"
	"testing"

	"github.com/mgwinsor/meridian/backend/internal/currency"
)

func TestNew(t *testing.T) {
	usd, _ := currency.Parse("USD")
	id := NewID()
	for _, kind := range []Kind{KindStock, KindETF, KindBond, KindMutualFund, KindCrypto} {
		got, err := New(id, kind, " \tBRK.B\u00a0", " Berkshire Hathaway ", usd)
		if err != nil || got.ID != id || got.Kind != kind || got.Symbol != "BRK.B" || got.Name != "Berkshire Hathaway" || got.QuoteCurrency != usd {
			t.Fatalf("New(%q) = %+v, %v", kind, got, err)
		}
	}
	for _, tc := range []struct {
		name                string
		kind                Kind
		symbol, displayName string
		code                currency.Code
		want                error
	}{
		{"missing kind", "", "AAPL", "Apple", usd, ErrInvalidKind},
		{"unknown kind", "property", "AAPL", "Apple", usd, ErrInvalidKind},
		{"case sensitive kind", "Stock", "AAPL", "Apple", usd, ErrInvalidKind},
		{"blank symbol", KindStock, " \u00a0", "Apple", usd, ErrInvalidSymbol},
		{"blank name", KindStock, "AAPL", "\n\t", usd, ErrInvalidName},
		{"zero currency", KindStock, "AAPL", "Apple", currency.Code{}, currency.ErrUnsupported},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := New(id, tc.kind, tc.symbol, tc.displayName, tc.code); !errors.Is(err, tc.want) {
				t.Fatalf("got %v, want %v", err, tc.want)
			}
		})
	}
	got, err := New(id, KindStock, "aapl", "Apple", usd)
	if err != nil || got.Symbol != "aapl" {
		t.Fatalf("symbol case changed: %+v, %v", got, err)
	}
}

func TestID(t *testing.T) {
	id := NewID()
	parsed, err := ParseID(id.String())
	if err != nil || parsed != id {
		t.Fatalf("ID round trip: %v", err)
	}
	if NewID() == id {
		t.Fatal("IDs must be distinct")
	}
	for _, value := range []string{"", "bad-id", "123E4567-E89B-42D3-A456-426614174000", strings.ReplaceAll(id.String(), "-", ""), "urn:uuid:" + id.String()} {
		if _, err := ParseID(value); !errors.Is(err, ErrInvalidID) {
			t.Errorf("ParseID(%q): %v", value, err)
		}
	}
}
