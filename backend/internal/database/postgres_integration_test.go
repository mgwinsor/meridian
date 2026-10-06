//go:build integration

package database_test

import (
	"context"
	"errors"
	"net/url"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/mgwinsor/meridian/backend/internal/account"
	"github.com/mgwinsor/meridian/backend/internal/cash"
	"github.com/mgwinsor/meridian/backend/internal/currency"
	"github.com/mgwinsor/meridian/backend/internal/database"
	"github.com/mgwinsor/meridian/backend/internal/instrument"
	"github.com/mgwinsor/meridian/backend/internal/money"
	"github.com/mgwinsor/meridian/backend/internal/position"
	"github.com/mgwinsor/meridian/backend/internal/price"
	"github.com/mgwinsor/meridian/backend/internal/property"
)

// Each run creates and drops only its own database, leaving application data alone.
// The supplied role needs CREATEDB (the Compose development role has this).
func migratedDatabase(t *testing.T) string {
	t.Helper()
	connectionURL := os.Getenv("TEST_DATABASE_URL")
	if connectionURL == "" {
		connectionURL = database.URL()
	}
	parsed, err := url.Parse(connectionURL)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Scheme != "postgres" && parsed.Scheme != "postgresql" {
		t.Fatal("test database connection must be a postgres:// URL")
	}
	name := "meridian_test_" + uuid.NewString()
	parsed.Path = "/" + name
	testURL := parsed.String()
	testConfig, err := pgx.ParseConfig(testURL)
	if err != nil {
		t.Fatal(err)
	}
	if testConfig.Database != name {
		t.Fatal("test URL query overrides isolated database name")
	}
	adminConfig, err := pgx.ParseConfig(connectionURL)
	if err != nil {
		t.Fatal(err)
	}
	adminConfig.Database = "postgres"
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		db := stdlib.OpenDB(*adminConfig)
		defer db.Close()
		if _, err := db.ExecContext(ctx, "DROP DATABASE IF EXISTS "+pgx.Identifier{name}.Sanitize()+" WITH (FORCE)"); err != nil {
			t.Error(err)
		}
	})
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	if err := database.Create(ctx, testURL); err != nil {
		t.Fatal(err)
	}
	if err := database.Create(ctx, testURL); err != nil {
		t.Fatalf("repeated creation: %v", err)
	}
	config, err := pgx.ParseConfig(testURL)
	if err != nil {
		t.Fatal(err)
	}
	db := stdlib.OpenDB(*config)
	defer db.Close()
	provider, err := database.MigrationProvider(db)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := provider.Up(ctx); err != nil {
		t.Fatal(err)
	}
	if results, err := provider.Up(ctx); err != nil || len(results) != 0 {
		t.Fatalf("repeated up = %v, %v", results, err)
	}
	if _, err := provider.DownTo(ctx, 0); err != nil {
		t.Fatal(err)
	}
	var tableExists bool
	if err := db.QueryRowContext(ctx, "SELECT to_regclass('accounts') IS NOT NULL").Scan(&tableExists); err != nil || tableExists {
		t.Fatalf("down left accounts table: %v, %v", tableExists, err)
	}
	if _, err := provider.Up(ctx); err != nil {
		t.Fatal(err)
	}
	pool, err := database.Open(ctx, testURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if err := database.RequireCurrentSchema(ctx, pool); err != nil {
		t.Fatal(err)
	}
	if _, err := provider.DownTo(ctx, 0); err != nil {
		t.Fatal(err)
	}
	if err := database.RequireCurrentSchema(ctx, pool); err == nil {
		t.Fatal("server accepted a rolled-back schema")
	}
	if _, err := provider.Up(ctx); err != nil {
		t.Fatal(err)
	}
	return testURL
}

func TestPostgresRepositories(t *testing.T) {
	testURL := migratedDatabase(t)
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	pool, err := database.Open(ctx, testURL)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if pool != nil {
			pool.Close()
		}
	}()

	accounts := account.NewPostgresRepository(pool)
	instruments := instrument.NewPostgresRepository(pool)
	balances := cash.NewPostgresRepository(pool)
	properties := property.NewPostgresRepository(pool)
	positions := position.NewPostgresRepository(pool)
	prices := price.NewPostgresRepository(pool)

	usd, _ := currency.Parse("USD")
	vnd, _ := currency.Parse("VND")
	maximum, _ := money.FromMinorUnits(usd, 9223372036854775807)
	otherValue, _ := money.FromMinorUnits(vnd, 123)
	a, _ := account.New(account.NewID(), "First")
	b, _ := account.New(account.NewID(), "Second")
	i, _ := instrument.New(instrument.NewID(), instrument.KindETF, "TEST", "Test fund", usd)
	p, _ := property.New(property.NewID(), "Home", maximum)

	if rows, err := accounts.List(ctx); err != nil || rows == nil || len(rows) != 0 {
		t.Fatalf("empty accounts = %v, %v", rows, err)
	}
	if rows, err := instruments.List(ctx); err != nil || rows == nil || len(rows) != 0 {
		t.Fatalf("empty instruments = %v, %v", rows, err)
	}
	if rows, err := properties.List(ctx); err != nil || rows == nil || len(rows) != 0 {
		t.Fatalf("empty properties = %v, %v", rows, err)
	}
	if _, err := accounts.FindByID(ctx, a.ID); !errors.Is(err, account.ErrNotFound) {
		t.Fatalf("missing account = %v", err)
	}
	if _, err := instruments.FindByID(ctx, i.ID); !errors.Is(err, instrument.ErrNotFound) {
		t.Fatalf("missing instrument = %v", err)
	}
	if _, err := properties.ReplaceValue(ctx, p.ID, maximum); !errors.Is(err, property.ErrNotFound) {
		t.Fatalf("missing property = %v", err)
	}
	for _, item := range []account.Account{a, b} {
		if err := accounts.Save(ctx, item); err != nil {
			t.Fatal(err)
		}
	}
	a.Name = "Renamed"
	if err := accounts.Save(ctx, a); err != nil {
		t.Fatal(err)
	}
	if err := instruments.Save(ctx, i); err != nil {
		t.Fatal(err)
	}
	i.Name = "Renamed fund"
	if err := instruments.Save(ctx, i); err != nil {
		t.Fatal(err)
	}
	if err := properties.Create(ctx, p); err != nil {
		t.Fatal(err)
	}

	if rows, err := balances.ListByAccount(ctx, a.ID); err != nil || rows == nil || len(rows) != 0 {
		t.Fatalf("empty cash = %v, %v", rows, err)
	}
	if rows, err := positions.ListByAccount(ctx, a.ID); err != nil || rows == nil || len(rows) != 0 {
		t.Fatalf("empty positions = %v, %v", rows, err)
	}
	if rows, err := prices.ListByInstrument(ctx, i.ID); err != nil || rows == nil || len(rows) != 0 {
		t.Fatalf("empty prices = %v, %v", rows, err)
	}
	zero, _ := money.FromMinorUnits(usd, 0)
	for _, amount := range []money.Amount{zero, maximum, otherValue} {
		if err := balances.Save(ctx, cash.Balance{AccountID: a.ID, Amount: amount}); err != nil {
			t.Fatal(err)
		}
	}
	if rows, err := balances.ListByAccount(ctx, b.ID); err != nil || len(rows) != 0 {
		t.Fatalf("cash leaked between accounts: %v, %v", rows, err)
	}

	// Exact maximum NUMERIC(38,18) value survives the pgx decimal codec.
	quantity, err := position.ParseQuantity("99999999999999999999.999999999999999999")
	if err != nil {
		t.Fatal(err)
	}
	holding := position.Position{AccountID: a.ID, InstrumentID: i.ID, Quantity: quantity}
	if err := positions.Save(ctx, holding); err != nil {
		t.Fatal(err)
	}
	if rows, err := positions.ListByAccount(ctx, a.ID); err != nil || len(rows) != 1 || !rows[0].Quantity.Equal(quantity) {
		t.Fatalf("exact quantity round trip failed: %v", err)
	}
	holding.Quantity = position.Quantity{}
	if err := positions.Save(ctx, holding); err != nil {
		t.Fatal(err)
	}
	if rows, err := positions.ListByAccount(ctx, b.ID); err != nil || len(rows) != 0 {
		t.Fatalf("positions leaked between accounts: %v, %v", rows, err)
	}

	// Replacements must return one whole currency/amount pair under contention.
	var wg sync.WaitGroup
	failures := make(chan error, 24)
	for n := 0; n < cap(failures); n++ {
		wg.Go(func() {
			value := maximum
			if n%2 == 0 {
				value = otherValue
			}
			updated, err := properties.ReplaceValue(ctx, p.ID, value)
			if err != nil {
				failures <- err
				return
			}
			if updated.Value != value || updated.Name != p.Name {
				failures <- errors.New("property replacement mixed fields")
			}
		})
	}
	wg.Wait()
	close(failures)
	for err := range failures {
		t.Error(err)
	}
	if _, err := properties.ReplaceValue(ctx, p.ID, otherValue); err != nil {
		t.Fatal(err)
	}
	p.Value = otherValue

	// Append semantics, stable equal-time ordering, microsecond truncation, and date bounds.
	instants := []time.Time{
		time.Date(9999, 12, 31, 23, 59, 59, 999999999, time.UTC),
		time.Date(0, 1, 1, 0, 0, 0, 1, time.UTC),
		time.Date(1, 1, 1, 0, 0, 0, 1000, time.UTC),
		time.Date(2026, 9, 28, 12, 0, 0, 123456789, time.FixedZone("offset", 3600)),
	}
	for _, instant := range instants {
		observation, err := price.New(i.ID, maximum, instant)
		if err != nil {
			t.Fatal(err)
		}
		if err := prices.Save(ctx, observation); err != nil {
			t.Fatal(err)
		}
	}
	duplicate, _ := price.New(i.ID, zero, instants[3])
	for n := 0; n < 2; n++ {
		if err := prices.Save(ctx, duplicate); err != nil {
			t.Fatal(err)
		}
	}

	// Closing all connections and rebuilding repositories exercises durable reads.
	pool.Close()
	pool, err = database.Open(ctx, testURL)
	if err != nil {
		t.Fatal(err)
	}
	accounts = account.NewPostgresRepository(pool)
	instruments = instrument.NewPostgresRepository(pool)
	balances = cash.NewPostgresRepository(pool)
	properties = property.NewPostgresRepository(pool)
	positions = position.NewPostgresRepository(pool)
	prices = price.NewPostgresRepository(pool)
	if got, err := accounts.FindByID(ctx, a.ID); err != nil || got != a {
		t.Fatalf("account = %v, %v", got, err)
	}
	if got, err := instruments.FindByID(ctx, i.ID); err != nil || got != i {
		t.Fatalf("instrument = %v, %v", got, err)
	}
	if got, err := properties.List(ctx); err != nil || len(got) != 1 || got[0] != p {
		t.Fatalf("property = %v, %v", got, err)
	}
	if got, err := balances.ListByAccount(ctx, a.ID); err != nil || len(got) != 2 || got[0].Amount != maximum || got[1].Amount != otherValue {
		t.Fatalf("cash = %v, %v", got, err)
	}
	if got, err := positions.ListByAccount(ctx, a.ID); err != nil || len(got) != 1 || got[0].Quantity.String() != "0" {
		t.Fatalf("position = %v, %v", got, err)
	}
	gotPrices, err := price.NewService(instruments, prices).ListObservations(ctx, i.ID)
	if err != nil || len(gotPrices) != 6 {
		t.Fatalf("prices = %v, %v", gotPrices, err)
	}
	for n, want := range []time.Time{instants[1], instants[2], instants[3], instants[3], instants[3], instants[0]} {
		if !gotPrices[n].ObservedAt.Equal(want.Truncate(time.Microsecond)) || gotPrices[n].ObservedAt.Location() != time.UTC {
			t.Fatalf("timestamp %d = %v, want %v", n, gotPrices[n].ObservedAt, want)
		}
	}
	if gotPrices[2].Amount != maximum || gotPrices[3].Amount != zero || gotPrices[4].Amount != zero {
		t.Fatal("equal-time insertion order changed")
	}

	// Foreign keys and checks remain authoritative for writes outside the services.
	expectPGError := func(err error, code string) {
		t.Helper()
		var pgErr *pgconn.PgError
		if !errors.As(err, &pgErr) || pgErr.Code != code {
			t.Fatalf("database error = %v, want %s", err, code)
		}
	}
	expectPGError(balances.Save(ctx, cash.Balance{AccountID: account.NewID(), Amount: maximum}), "23503")
	expectPGError(positions.Save(ctx, position.Position{AccountID: a.ID, InstrumentID: instrument.NewID()}), "23503")
	orphan, _ := price.New(instrument.NewID(), maximum, instants[3])
	expectPGError(prices.Save(ctx, orphan), "23503")
	_, err = pool.Exec(ctx, "UPDATE cash_balances SET minor_units = -1 WHERE account_id = $1", a.ID.String())
	expectPGError(err, "23514")
	_, err = pool.Exec(ctx, "UPDATE cash_balances SET currency = 'BAD' WHERE account_id = $1", a.ID.String())
	expectPGError(err, "23514")
	canceled, cancelRequest := context.WithCancel(ctx)
	cancelRequest()
	if _, err := accounts.List(canceled); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled read = %v", err)
	}
}
