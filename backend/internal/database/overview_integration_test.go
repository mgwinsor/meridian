//go:build integration

package database_test

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mgwinsor/meridian/backend/internal/account"
	"github.com/mgwinsor/meridian/backend/internal/assetclass"
	"github.com/mgwinsor/meridian/backend/internal/cash"
	"github.com/mgwinsor/meridian/backend/internal/currency"
	"github.com/mgwinsor/meridian/backend/internal/database"
	"github.com/mgwinsor/meridian/backend/internal/fx"
	"github.com/mgwinsor/meridian/backend/internal/instrument"
	"github.com/mgwinsor/meridian/backend/internal/money"
	"github.com/mgwinsor/meridian/backend/internal/overview"
	"github.com/mgwinsor/meridian/backend/internal/position"
	"github.com/mgwinsor/meridian/backend/internal/price"
	"github.com/mgwinsor/meridian/backend/internal/property"
)

func TestInstrumentClassification(t *testing.T) {
	url := migratedDatabase(t)
	ctx := t.Context()
	pool, err := database.Open(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	repo := instrument.NewPostgresRepository(pool)
	usd, _ := currency.Parse("USD")
	for kind, want := range map[instrument.Kind]assetclass.Class{
		instrument.KindStock:      assetclass.Equities,
		instrument.KindBond:       assetclass.Bonds,
		instrument.KindCrypto:     assetclass.Crypto,
		instrument.KindETF:        assetclass.Unclassified,
		instrument.KindMutualFund: assetclass.Unclassified,
	} {
		t.Run(string(kind), func(t *testing.T) {
			item, err := instrument.New(instrument.NewID(), kind, "TEST", "Original", usd)
			if err != nil {
				t.Fatal(err)
			}
			if err := repo.Save(ctx, item); err != nil {
				t.Fatal(err)
			}
			stored, err := repo.FindByID(ctx, item.ID)
			if err != nil || stored.AssetClass != want {
				t.Fatalf("class round trip: %+v %v", stored, err)
			}

			// A class-only write preserves another writer's committed metadata.
			_, err = pool.Exec(ctx, "UPDATE instruments SET name='Renamed' WHERE id=$1", item.ID.String())
			if err != nil {
				t.Fatal(err)
			}
			stored, err = repo.ReplaceAssetClass(ctx, item.ID, assetclass.Mixed)
			if err != nil || stored.Name != "Renamed" || stored.AssetClass != assetclass.Mixed {
				t.Fatalf("class update: %+v %v", stored, err)
			}
			stored, err = repo.FindByID(ctx, item.ID)
			if err != nil || stored.AssetClass != assetclass.Mixed {
				t.Fatalf("custom class persistence: %+v %v", stored, err)
			}

			for _, tc := range []struct {
				value any
				code  string
			}{{nil, "23502"}, {"cash", "23514"}, {"property", "23514"}, {"unknown", "23514"}} {
				_, err := pool.Exec(ctx,
					"UPDATE instruments SET asset_class=$1 WHERE id=$2", tc.value, item.ID.String())
				var pgErr *pgconn.PgError
				if !errors.As(err, &pgErr) || pgErr.Code != tc.code {
					t.Fatalf("invalid class %v: %v", tc.value, err)
				}
			}
		})
	}
}

type snapshotBarrier struct {
	reached, resume chan struct{}
	once            atomic.Bool
}
type snapshotQueryKey struct{}

func (b *snapshotBarrier) TraceQueryStart(
	ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryStartData,
) context.Context {
	return context.WithValue(ctx, snapshotQueryKey{}, strings.Contains(data.SQL, "FROM accounts ORDER BY id"))
}

func (b *snapshotBarrier) TraceQueryEnd(ctx context.Context, _ *pgx.Conn, _ pgx.TraceQueryEndData) {
	if yes, _ := ctx.Value(snapshotQueryKey{}).(bool); yes && b.once.CompareAndSwap(false, true) {
		close(b.reached)
		select {
		case <-b.resume:
		case <-ctx.Done():
		}
	}
}

func TestOverviewSnapshotAndBulkReads(t *testing.T) {
	url := migratedDatabase(t)
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	pool, err := database.Open(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	usd, _ := currency.Parse("USD")
	sgd, _ := currency.Parse("SGD")
	vnd, _ := currency.Parse("VND")
	a, _ := account.New(account.NewID(), "Broker")
	accounts := account.NewPostgresRepository(pool)
	if err = accounts.Save(ctx, a); err != nil {
		t.Fatal(err)
	}
	i, _ := instrument.New(instrument.NewID(), instrument.KindStock, "EQ", "Equity", sgd)
	instruments := instrument.NewPostgresRepository(pool)
	if err = instruments.Save(ctx, i); err != nil {
		t.Fatal(err)
	}
	cashRepo := cash.NewPostgresRepository(pool)
	for _, code := range []currency.Code{usd, sgd} {
		amount, _ := money.Parse(code, "100")
		if err = cashRepo.Save(ctx, cash.Balance{AccountID: a.ID, Amount: amount}); err != nil {
			t.Fatal(err)
		}
	}
	amount, _ := money.Parse(vnd, "20000")
	p, _ := property.New(property.NewID(), "Home", amount)
	if err = property.NewPostgresRepository(pool).Create(ctx, p); err != nil {
		t.Fatal(err)
	}
	q, _ := position.ParseQuantity("2")
	holding := position.Position{AccountID: a.ID, InstrumentID: i.ID, Quantity: q}
	if err = position.NewPostgresRepository(pool).Save(ctx, holding); err != nil {
		t.Fatal(err)
	}
	at := time.Date(2026, 1, 1, 0, 0, 0, 123456000, time.UTC)
	prices := price.NewPostgresRepository(pool)
	rates := fx.NewPostgresRepository(pool)
	for _, item := range []struct {
		amount string
		at     time.Time
	}{{"5", at}, {"999", at.Add(time.Hour)}, {"1", at.Add(-time.Hour)}, {"10", at}} {
		amount, _ := money.Parse(sgd, item.amount)
		observation, _ := price.New(i.ID, amount, item.at)
		if err = prices.Save(ctx, observation); err != nil {
			t.Fatal(err)
		}
	}
	for _, item := range []struct {
		code currency.Code
		rate string
		at   time.Time
	}{
		{sgd, "1", at},
		{sgd, "999", at.Add(time.Hour)},
		{sgd, "0.5", at.Add(-time.Hour)},
		{sgd, "2", at},
		{vnd, "20000", at},
	} {
		rate, _ := fx.ParseRate(item.rate)
		observation, _ := fx.New(item.code, rate, item.at)
		if err = rates.Save(ctx, observation); err != nil {
			t.Fatal(err)
		}
	}
	// The cutoff is inclusive; ties choose last inserted, backdated/future rows remain in history.
	latest, err := prices.Latest(ctx, []instrument.ID{i.ID, instrument.NewID()}, at)
	if err != nil || len(latest) != 1 || latest[0].Amount.String() != "10.00" {
		t.Fatalf("prices: %+v %v", latest, err)
	}
	missing, err := prices.Latest(ctx, []instrument.ID{i.ID}, at.Add(-2*time.Hour))
	if err != nil || missing == nil || len(missing) != 0 {
		t.Fatalf("missing: %+v %v", missing, err)
	}
	latestRates, err := rates.Latest(ctx, at)
	if err != nil || len(latestRates) != 2 || latestRates[0].Rate.String() != "2" {
		t.Fatalf("rates: %+v %v", latestRates, err)
	}
	history, err := rates.List(ctx, sgd)
	if err != nil || len(history) != 4 || history[1].Rate.String() != "1" || history[2].Rate.String() != "2" {
		t.Fatalf("history: %+v %v", history, err)
	}
	before, err := rates.Latest(ctx, at.Add(-2*time.Hour))
	if err != nil || before == nil || len(before) != 0 {
		t.Fatalf("missing rates: %+v %v", before, err)
	}
	// Exact maximum rate survives PostgreSQL NUMERIC independently of minor units.
	maximum, _ := fx.ParseRate("99999999999999999999.999999999999999999")
	observation, _ := fx.New(vnd, maximum, at.Add(2*time.Hour))
	if err = rates.Save(ctx, observation); err != nil {
		t.Fatal(err)
	}
	persisted, err := rates.Latest(ctx, at.Add(2*time.Hour))
	if err != nil || persisted[1].Rate.String() != maximum.String() {
		t.Fatalf("exact FX round trip: %+v %v", persisted, err)
	}
	barrier := &snapshotBarrier{reached: make(chan struct{}), resume: make(chan struct{})}
	config, err := pgxpool.ParseConfig(url)
	if err != nil {
		t.Fatal(err)
	}
	config.ConnConfig.Tracer = barrier
	readPool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	defer readPool.Close()
	type result struct {
		input overview.Inputs
		err   error
	}
	done := make(chan result, 1)
	go func() {
		input, err := overview.NewPostgresReader(readPool).Read(ctx, at)
		done <- result{input, err}
	}()
	select {
	case <-barrier.reached:
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	// Commit changes between account and cash/instrument reads. All overview inputs must stay old.
	tx, err := pool.Begin(ctx)
	if err != nil {
		close(barrier.resume)
		t.Fatal(err)
	}
	a.Name = "Changed"
	err = account.NewPostgresRepository(tx).Save(ctx, a)
	if err == nil {
		amount, _ := money.Parse(usd, "200")
		err = cash.NewPostgresRepository(tx).Save(ctx, cash.Balance{AccountID: a.ID, Amount: amount})
	}
	if err == nil {
		_, err = instrument.NewPostgresRepository(tx).ReplaceAssetClass(ctx, i.ID, assetclass.Mixed)
	}
	if err == nil {
		err = tx.Commit(ctx)
	} else {
		_ = tx.Rollback(ctx)
	}
	close(barrier.resume)
	if err != nil {
		t.Fatal(err)
	}
	snapshot := <-done
	if snapshot.err != nil {
		t.Fatal(snapshot.err)
	}
	if snapshot.input.Accounts[0].Name != "Broker" ||
		snapshot.input.Instruments[0].AssetClass != assetclass.Equities ||
		len(snapshot.input.Cash) != 2 || len(snapshot.input.Positions) != 1 {
		t.Fatalf("mixed snapshot: %+v", snapshot.input)
	}
	for _, tc := range []struct {
		code currency.Code
		want string
	}{{usd, "161.00"}, {sgd, "322.00"}, {vnd, "3220000"}} {
		r, err := overview.Calculate(snapshot.input, tc.code, at)
		if err != nil || !r.Complete || r.ValuedTotal != tc.want {
			t.Fatalf("fixture: %+v %v", r, err)
		}
	}
	current, err := overview.NewPostgresReader(pool).Read(ctx, at)
	if err != nil {
		t.Fatal(err)
	}
	r, err := overview.Calculate(current, usd, at)
	if err != nil || r.ValuedTotal != "261.00" || current.Instruments[0].AssetClass != assetclass.Mixed {
		t.Fatalf("new snapshot: %+v %v", r, err)
	}
}
