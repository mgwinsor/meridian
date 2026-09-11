package currency_test

import (
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
