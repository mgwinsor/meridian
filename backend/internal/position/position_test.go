package position

import (
	"errors"
	"testing"
)

func TestParseQuantity(t *testing.T) {
	for input, want := range map[string]string{
		"00099999999999999999999.999999999999999999000": "99999999999999999999.999999999999999999",
		"0": "0", "0.000": "0", " 0012.3400 ": "12.34",
		"0.000000000000000001":                    "0.000000000000000001",
		"99999999999999999999.999999999999999999": "99999999999999999999.999999999999999999",
	} {
		got, err := ParseQuantity(input)
		if err != nil || got.String() != want {
			t.Errorf("ParseQuantity(%q) = %q, %v; want %q", input, got.String(), err, want)
		}
	}
	for _, input := range []string{"", " ", "-1", "-0", "+1", ".1", "1.", "1.2.3", "1e3", "NaN", "Infinity", "1,000", "1 2", "１２", "100000000000000000000", "0.0000000000000000001", "1.1234567890123456789", "99999999999999999999.9999999999999999999"} {
		if _, err := ParseQuantity(input); !errors.Is(err, ErrInvalidQuantity) {
			t.Errorf("ParseQuantity(%q) error = %v", input, err)
		}
	}
	if (Quantity{}).String() != "0" {
		t.Fatal("zero value must represent zero")
	}
}

func TestQuantityEqual(t *testing.T) {
	one, err := ParseQuantity("1.0")
	if err != nil {
		t.Fatalf("ParseQuantity() unexpected error: %v", err)
	}
	same, err := ParseQuantity("01.000")
	if err != nil {
		t.Fatalf("ParseQuantity() unexpected error: %v", err)
	}
	different, err := ParseQuantity("1.01")
	if err != nil {
		t.Fatalf("ParseQuantity() unexpected error: %v", err)
	}

	if !one.Equal(same) {
		t.Error("numerically equivalent quantities should be equal")
	}
	if one.Equal(different) {
		t.Error("different quantities should not be equal")
	}
}
