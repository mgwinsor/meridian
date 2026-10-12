package overview

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/mgwinsor/meridian/backend/internal/account"
	"github.com/mgwinsor/meridian/backend/internal/assetclass"
	"github.com/mgwinsor/meridian/backend/internal/cash"
	"github.com/mgwinsor/meridian/backend/internal/currency"
	"github.com/mgwinsor/meridian/backend/internal/fx"
	"github.com/mgwinsor/meridian/backend/internal/instrument"
	"github.com/mgwinsor/meridian/backend/internal/money"
	"github.com/mgwinsor/meridian/backend/internal/position"
	"github.com/mgwinsor/meridian/backend/internal/price"
	"github.com/mgwinsor/meridian/backend/internal/property"
	"github.com/shopspring/decimal"
)

var instant = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

func code(s string) currency.Code { c, _ := currency.Parse(s); return c }

func amount(c, s string) money.Amount {
	a, err := money.Parse(code(c), s)
	if err != nil {
		panic(err)
	}
	return a
}

func quantity(s string) position.Quantity {
	q, err := position.ParseQuantity(s)
	if err != nil {
		panic(err)
	}
	return q
}

func rate(c, s string) fx.Observation {
	r, _ := fx.ParseRate(s)
	o, err := fx.New(code(c), r, instant)
	if err != nil {
		panic(err)
	}
	return o
}

func fixture() Inputs {
	a, _ := account.New(account.NewID(), "Broker")
	i, _ := instrument.New(instrument.NewID(), instrument.KindStock, "TEST", "Equity", code("SGD"))
	p, _ := property.New(property.NewID(), "Home", amount("VND", "20000"))
	observation, _ := price.New(i.ID, amount("SGD", "10"), instant)
	return Inputs{
		Accounts:    []account.Account{a},
		Instruments: []instrument.Instrument{i},
		Cash: []cash.Balance{
			{AccountID: a.ID, Amount: amount("USD", "100")},
			{AccountID: a.ID, Amount: amount("SGD", "100")},
		},
		Properties: []property.Property{p},
		Positions: []position.Position{
			{AccountID: a.ID, InstrumentID: i.ID, Quantity: quantity("2")},
		},
		Prices: []price.Observation{observation},
		Rates:  []fx.Observation{rate("SGD", "2"), rate("VND", "20000")},
	}
}

func reconcile(t *testing.T, r Report) {
	t.Helper()
	for _, rows := range [][]Breakdown{r.AssetClasses, r.Currencies} {
		sum := decimal.Zero
		for _, row := range rows {
			value, _ := decimal.NewFromString(row.Amount)
			sum = sum.Add(value)
		}
		total, _ := decimal.NewFromString(r.ValuedTotal)
		if !sum.Equal(total) {
			t.Fatalf("breakdown %v != %s", rows, r.ValuedTotal)
		}
	}
	sum := decimal.Zero
	for _, row := range r.Assets {
		if row.ConvertedValue != nil {
			value, _ := decimal.NewFromString(*row.ConvertedValue)
			sum = sum.Add(value)
		}
	}
	total, _ := decimal.NewFromString(r.ValuedTotal)
	if !sum.Equal(total) {
		t.Fatal("assets do not reconcile")
	}
}

func TestFixture(t *testing.T) {
	input := fixture()
	for c, want := range map[string]string{"USD": "161.00", "SGD": "322.00", "VND": "3220000"} {
		t.Run(c, func(t *testing.T) {
			r, err := Calculate(input, code(c), instant)
			if err != nil || !r.Complete || r.ValuedTotal != want || len(r.Assets) != 4 {
				t.Fatalf("report = %+v, %v", r, err)
			}
			reconcile(t, r)
			if r.Assets[2].AssetClass != assetclass.Equities {
				t.Fatal("holding classification")
			}
		})
	}
}

func TestMissingAndZero(t *testing.T) {
	input := fixture()
	input.Prices = nil
	input.Rates = nil
	r, err := Calculate(input, code("USD"), instant)
	if err != nil || r.Complete || r.ValuedTotal != "100.00" {
		t.Fatalf("%+v %v", r, err)
	}
	if r.Assets[2].NativeValue != nil || len(r.Assets[2].MissingInputs) != 1 || r.Assets[2].MissingInputs[0] != "price" {
		t.Fatal(r.Assets[2])
	}
	excluded := 0
	for _, row := range r.AssetClasses {
		excluded += row.ExcludedAssets
	}
	if excluded != 3 {
		t.Fatal(excluded)
	}
	reconcile(t, r)
	input.Cash = nil
	input.Properties = nil
	input.Positions[0].Quantity = quantity("0")
	r, err = Calculate(input, code("VND"), instant)
	if err != nil || !r.Complete || r.ValuedTotal != "0" ||
		len(r.FXObservations) != 0 || r.AssetClasses[0].Percentage != nil {
		t.Fatalf("%+v %v", r, err)
	}
	// A zero price also makes an unconverted foreign holding known-zero.
	input.Positions[0].Quantity = quantity("123")
	observation, _ := price.New(input.Instruments[0].ID, amount("SGD", "0"), instant)
	input.Prices = []price.Observation{observation}
	r, err = Calculate(input, code("VND"), instant)
	if err != nil || !r.Complete || r.ValuedTotal != "0" {
		t.Fatalf("%+v %v", r, err)
	}
	// Same-currency amounts require no rate, including target rate.
	input = fixture()
	input.Cash = input.Cash[1:]
	input.Properties = nil
	input.Positions = nil
	input.Rates = nil
	r, err = Calculate(input, code("SGD"), instant)
	if err != nil || !r.Complete || r.ValuedTotal != "100.00" {
		t.Fatalf("%+v %v", r, err)
	}
}

func TestExactRoundingAndLargeTotals(t *testing.T) {
	input := fixture()
	input.Cash = nil
	input.Properties = nil
	for _, tc := range []struct{ q, p, source, target, want string }{
		{"0.001", "10", "2", "USD", "0.01"}, // exact half cent, round up
		{"0.000999999999999999", "10", "2", "USD", "0.00"},
		{"0.1", "10", "3", "USD", "0.33"},
		{"0.1", "10", "2", "VND", "10000"},
		{
			q:      "99999999999999999999.999999999999999999",
			p:      "92233720368547758.07",
			source: "1",
			target: "SGD",
			want:   "9223372036854775806999999999999999999.91",
		},
	} {
		input.Positions[0].Quantity = quantity(tc.q)
		observation, _ := price.New(input.Instruments[0].ID, amount("SGD", tc.p), instant)
		input.Prices = []price.Observation{observation}
		input.Rates = []fx.Observation{rate("SGD", tc.source), rate("VND", "20000")}
		r, err := Calculate(input, code(tc.target), instant)
		if err != nil || r.ValuedTotal != tc.want {
			t.Fatalf("%+v: got %s err %v", tc, r.ValuedTotal, err)
		}
		reconcile(t, r)
	}
	input = fixture()
	input.Cash = []cash.Balance{
		{AccountID: account.NewID(), Amount: amount("USD", "92233720368547758.07")},
		{AccountID: account.NewID(), Amount: amount("USD", "92233720368547758.07")},
	}
	input.Properties = nil
	input.Positions = nil
	r, _ := Calculate(input, code("USD"), instant)
	if r.ValuedTotal != "184467440737095516.14" {
		t.Fatal(r.ValuedTotal)
	}
	reconcile(t, r)
	// Round each asset before summing (two half cents become two cents).
	input.Cash = []cash.Balance{
		{AccountID: account.NewID(), Amount: amount("SGD", "0.01")},
		{AccountID: account.NewID(), Amount: amount("SGD", "0.01")},
	}
	r, _ = Calculate(input, code("USD"), instant)
	if r.ValuedTotal != "0.02" {
		t.Fatal(r.ValuedTotal)
	}
	reconcile(t, r)
}

type readerFunc func(context.Context, time.Time) (Inputs, error)

func (f readerFunc) Read(ctx context.Context, at time.Time) (Inputs, error) { return f(ctx, at) }

func TestHTTP(t *testing.T) {
	mux := http.NewServeMux()
	reader := readerFunc(func(context.Context, time.Time) (Inputs, error) {
		return Inputs{}, nil
	})
	NewHandler(NewService(reader)).RegisterRoutes(mux)
	for _, path := range []string{"/api/v1/overview", "/api/v1/overview?currency=VND"} {
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		var r Report
		if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &r) != nil || !r.Complete ||
			r.Assets == nil || r.AssetClasses == nil || r.Currencies == nil || r.FXObservations == nil {
			t.Fatal(w.Body.String())
		}
	}
	for _, c := range []string{"", "usd", "EUR"} {
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/overview?currency="+c, nil))
		if w.Code != 400 {
			t.Fatal(w.Code)
		}
	}
	mux = http.NewServeMux()
	reader = readerFunc(func(context.Context, time.Time) (Inputs, error) {
		return Inputs{}, errors.New("private storage error")
	})
	NewHandler(NewService(reader)).RegisterRoutes(mux)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/overview", nil))
	if w.Code != 500 || w.Body.String() != "internal server error\n" {
		t.Fatal(w.Body.String())
	}
}
