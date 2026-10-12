package position

import (
	"context"
	"testing"

	"github.com/mgwinsor/meridian/backend/internal/account"
	"github.com/mgwinsor/meridian/backend/internal/instrument"
)

func TestWorkspaceList(t *testing.T) {
	repo := NewMemoryRepository()
	ctx := context.Background()
	rows, err := repo.List(ctx)
	if err != nil || rows == nil || len(rows) != 0 {
		t.Fatalf("empty: %v %v", rows, err)
	}
	first, second := account.NewID(), account.NewID()
	id := instrument.NewID()
	q, _ := ParseQuantity("0.123456789123456789")
	for _, a := range []account.ID{first, second} {
		if err := repo.Save(ctx, Position{AccountID: a, InstrumentID: id, Quantity: q}); err != nil {
			t.Fatal(err)
		}
	}
	if err := repo.Save(ctx, Position{AccountID: first, InstrumentID: id}); err != nil {
		t.Fatal(err)
	}
	rows, err = repo.List(ctx)
	if err != nil || len(rows) != 2 {
		t.Fatalf("workspace: %v %v", rows, err)
	}
	values := map[account.ID]string{}
	for _, row := range rows {
		values[row.AccountID] = row.Quantity.String()
	}
	if values[first] != "0" || values[second] != q.String() {
		t.Fatal(values)
	}
	rows[0].Quantity = q
	scoped, _ := repo.ListByAccount(ctx, first)
	if scoped[0].Quantity.String() != "0" {
		t.Fatal("list leaked storage")
	}
}
