package currency_test

import (
	"errors"
	"testing"

	"github.com/mgwinsor/meridian/backend/internal/currency"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		want      string
		wantError bool
	}{
		{
			name:  "USD",
			value: "USD",
			want:  "USD",
		},
		{
			name:  "normalizes lowercase",
			value: "sgd",
			want:  "SGD",
		},
		{
			name:  "strips whitespace",
			value: " VND ",
			want:  "VND",
		},
		{
			name:      "unsupported currency",
			value:     "GBP",
			wantError: true,
		},
		{
			name:      "invalid code",
			value:     "foo",
			wantError: true,
		},
		{
			name:      "empty",
			value:     "",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := currency.Parse(tt.value)

			if tt.wantError {
				if err == nil {
					t.Fatal("currency.Parse() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("currency.Parse() unexpected error: %v", err)
			}

			if got.String() != tt.want {
				t.Errorf(
					"Code.String() = %q, want %q",
					got.String(),
					tt.want,
				)
			}
		})
	}
}

func TestMinorUnitDigits(t *testing.T) {
	tests := []struct {
		code string
		want int
	}{
		{code: "USD", want: 2},
		{code: "SGD", want: 2},
		{code: "VND", want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			code, err := currency.Parse(tt.code)
			if err != nil {
				t.Fatalf("currency.Parse() unexpected error: %v", err)
			}

			got, err := code.MinorUnitDigits()
			if err != nil {
				t.Fatalf("Code.MinorUnitDigits() unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("Code.MinorUnitDigits() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestZeroCodeHasNoMinorUnitDigits(t *testing.T) {
	var code currency.Code

	_, err := code.MinorUnitDigits()
	if !errors.Is(err, currency.ErrUnsupported) {
		t.Fatalf("Code.MinorUnitDigits() error = %v, want ErrUnsupported", err)
	}
}
