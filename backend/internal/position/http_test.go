package position

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mgwinsor/meridian/backend/internal/account"
	"github.com/mgwinsor/meridian/backend/internal/instrument"
)

func TestHTTPWorkflow(t *testing.T) {
	ctx := context.Background()
	accounts := account.NewMemoryRepository()
	instruments := instrument.NewMemoryRepository()
	a, other := account.NewID(), account.NewID()
	i, j := instrument.NewID(), instrument.NewID()
	for _, id := range []account.ID{a, other} {
		if err := accounts.Save(ctx, account.Account{ID: id}); err != nil {
			t.Fatal(err)
		}
	}
	for _, id := range []instrument.ID{i, j} {
		if err := instruments.Save(ctx, instrument.Instrument{ID: id}); err != nil {
			t.Fatal(err)
		}
	}
	repository := NewMemoryRepository()
	mux := http.NewServeMux()
	NewHandler(NewService(accounts, instruments, repository)).RegisterRoutes(mux)
	base := "/api/v1/accounts/" + a.String() + "/positions"
	request := func(method, path, body string, status int) string {
		t.Helper()
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, httptest.NewRequest(method, path, strings.NewReader(body)))
		if w.Code != status {
			t.Fatalf("%s %s: %d %s; want %d", method, path, w.Code, w.Body.String(), status)
		}
		return w.Body.String()
	}
	if got := request("GET", base, "", 200); got != "{\"positions\":[]}\n" {
		t.Fatal(got)
	}
	for _, id := range []instrument.ID{i, j} {
		request("PUT", base+"/"+id.String(), `{"quantity":"001.23456789012345678900"}`, 200)
	}
	var result positionResponse
	if err := json.Unmarshal([]byte(request("PUT", base+"/"+i.String(), `{"quantity":"0"}`, 200)), &result); err != nil {
		t.Fatal(err)
	}
	if result.AccountID != a.String() || result.InstrumentID != i.String() || result.Quantity != "0" {
		t.Fatalf("response = %+v", result)
	}
	var list positionsResponse
	if err := json.Unmarshal([]byte(request("GET", base, "", 200)), &list); err != nil {
		t.Fatal(err)
	}
	if len(list.Positions) != 2 || list.Positions[0].InstrumentID >= list.Positions[1].InstrumentID {
		t.Fatalf("positions = %+v", list)
	}
	for _, p := range list.Positions {
		if p.InstrumentID == j.String() && p.Quantity != "1.234567890123456789" {
			t.Fatal(p)
		}
	}
	if got := request("GET", "/api/v1/accounts/"+other.String()+"/positions", "", 200); got != "{\"positions\":[]}\n" {
		t.Fatal(got)
	}
	for _, body := range []string{`{"quantity":"-1"}`, `{"quantity":1}`, `{"quantity":null}`, `{}`, `null`, `[]`, `{`} {
		request("PUT", base+"/"+i.String(), body, 400)
	}
	request("PUT", base+"/bad", `{"quantity":"1"}`, 400)
	request("GET", "/api/v1/accounts/bad/positions", "", 400)
	request("PUT", base+"/"+instrument.NewID().String(), `{"quantity":"1"}`, 404)
	missing := "/api/v1/accounts/" + account.NewID().String() + "/positions"
	request("GET", missing, "", 404)
	request("PUT", missing+"/"+i.String(), `{"quantity":"1"}`, 404)
	// Failed writes preserve the prior holding, and returned snapshots cannot mutate storage.
	snapshot, _ := repository.ListByAccount(ctx, a)
	if len(snapshot) != 2 {
		t.Fatal(snapshot)
	}
	snapshot[0].AccountID = other
	again, _ := repository.ListByAccount(ctx, a)
	for _, p := range again {
		if p.AccountID != a || (p.InstrumentID == i && p.Quantity.String() != "0") {
			t.Fatal(p)
		}
	}
}

func TestHTTPInternalError(t *testing.T) {
	mux := http.NewServeMux()
	NewHandler(NewService(existingAccountFinder(), existingInstrumentFinder(), &stubRepository{
		save:          func(context.Context, Position) error { return context.Canceled },
		listByAccount: func(context.Context, account.ID) ([]Position, error) { return nil, context.Canceled },
	})).RegisterRoutes(mux)
	base := "/api/v1/accounts/" + account.NewID().String() + "/positions"
	for _, method := range []string{"GET", "PUT"} {
		path := base
		if method == "PUT" {
			path += "/" + instrument.NewID().String()
		}
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, httptest.NewRequest(method, path, strings.NewReader(`{"quantity":"1"}`)))
		if w.Code != 500 || w.Body.String() != "internal server error\n" {
			t.Fatalf("%d %s", w.Code, w.Body.String())
		}
	}
}
