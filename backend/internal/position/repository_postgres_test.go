package position

import (
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/shopspring/decimal"
)

func TestPostgresNumericCodec(t *testing.T) {
	types := pgtype.NewMap()
	for _, input := range []string{"0", "0.000000000000000001", "99999999999999999999.999999999999999999"} {
		quantity, err := ParseQuantity(input)
		if err != nil {
			t.Fatal(err)
		}
		encoded, err := types.Encode(pgtype.NumericOID, pgtype.BinaryFormatCode, quantity.value, nil)
		if err != nil {
			t.Fatal(err)
		}
		var decoded decimal.Decimal
		if err := types.Scan(pgtype.NumericOID, pgtype.BinaryFormatCode, encoded, &decoded); err != nil {
			t.Fatal(err)
		}
		if !decoded.Equal(quantity.value) {
			t.Fatalf("numeric round trip = %s, want %s", decoded, input)
		}
	}
}
