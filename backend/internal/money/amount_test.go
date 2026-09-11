package money_test

import (
	"errors"
	"testing"

	"github.com/mgwinsor/meridian/backend/internal/currency"
	"github.com/mgwinsor/meridian/backend/internal/money"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name      string
		currency  string
		value     string
		want      string
		wantError bool
	}{
		{name: "two decimal places", currency: "USD", value: "1234.56", want: "1234.56"},
		{name: "pads fractional part", currency: "SGD", value: "10.5", want: "10.50"},
		{name: "pads whole amount", currency: "USD", value: "10", want: "10.00"},
		{name: "formats subunit", currency: "USD", value: "0.05", want: "0.05"},
		{name: "zero decimal currency", currency: "VND", value: "1000", want: "1000"},
		{name: "trims whitespace", currency: "USD", value: " 1.25 ", want: "1.25"},
		{name: "canonicalizes leading zeros", currency: "USD", value: "0001.20", want: "1.20"},
		{name: "zero", currency: "USD", value: "0", want: "0.00"},
		{name: "negative", currency: "USD", value: "-1.00", wantError: true},
		{name: "negative zero", currency: "USD", value: "-0.00", wantError: true},
		{name: "too many fractional digits", currency: "USD", value: "1.001", wantError: true},
		{name: "fraction for zero decimal currency", currency: "VND", value: "1.0", wantError: true},
		{name: "missing whole part", currency: "USD", value: ".50", wantError: true},
		{name: "missing fractional part", currency: "USD", value: "1.", wantError: true},
		{name: "plus sign", currency: "USD", value: "+1.00", wantError: true},
		{name: "non digits", currency: "USD", value: "one", wantError: true},
		{name: "multiple separators", currency: "USD", value: "1.2.3", wantError: true},
		{name: "empty", currency: "USD", value: "", wantError: true},
		{name: "overflow", currency: "USD", value: "92233720368547758.08", wantError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, err := currency.Parse(tt.currency)
			if err != nil {
				t.Fatalf("currency.Parse() unexpected error: %v", err)
			}

			got, err := money.Parse(code, tt.value)
			if tt.wantError {
				if !errors.Is(err, money.ErrInvalidAmount) {
					t.Fatalf("money.Parse() error = %v, want ErrInvalidAmount", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("money.Parse() unexpected error: %v", err)
			}
			if got.Currency != code {
				t.Errorf("Amount.Currency = %s, want %s", got.Currency, code)
			}
			if got.String() != tt.want {
				t.Errorf("Amount.String() = %q, want %q", got.String(), tt.want)
			}
		})
	}
}

func TestParseRejectsZeroCurrency(t *testing.T) {
	var code currency.Code

	_, err := money.Parse(code, "1")
	if !errors.Is(err, money.ErrInvalidAmount) {
		t.Fatalf("money.Parse() error = %v, want ErrInvalidAmount", err)
	}
}
