package property

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/mgwinsor/meridian/backend/internal/currency"
	"github.com/mgwinsor/meridian/backend/internal/money"
)

type failingRepository struct {
	calls int
	err   error
}

func (r *failingRepository) Create(context.Context, Property) error { r.calls++; return r.err }
func (r *failingRepository) List(context.Context) ([]Property, error) {
	r.calls++
	return nil, r.err
}
func (r *failingRepository) ReplaceValue(context.Context, ID, money.Amount) (Property, error) {
	r.calls++
	return Property{}, r.err
}

func parseAmount(t *testing.T, code, amount string) money.Amount {
	t.Helper()
	currency, err := currency.Parse(code)
	if err != nil {
		t.Fatal(err)
	}
	value, err := money.Parse(currency, amount)
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func TestServiceDependencyErrors(t *testing.T) {
	ctx := context.Background()
	repositoryError := errors.New("dependency unavailable")
	repository := &failingRepository{err: repositoryError}
	service := NewService(repository)
	amount := parseAmount(t, "USD", "1")

	if _, err := service.CreateProperty(ctx, "Home", amount); !errors.Is(err, repositoryError) {
		t.Fatalf("create error = %v", err)
	}
	if _, err := service.ListProperties(ctx); !errors.Is(err, repositoryError) {
		t.Fatalf("list error = %v", err)
	}
	if _, err := service.SetValue(ctx, NewID(), amount); !errors.Is(err, repositoryError) {
		t.Fatalf("update error = %v", err)
	}
	if repository.calls != 3 {
		t.Fatalf("repository calls = %d", repository.calls)
	}

	repository = &failingRepository{}
	service = NewService(repository)
	if _, err := service.CreateProperty(ctx, " \u0085 ", amount); !errors.Is(err, ErrInvalidName) {
		t.Fatalf("name error = %v", err)
	}
	if repository.calls != 0 {
		t.Fatal("invalid name saved")
	}
}

func TestInternalErrorsHaveSafeHTTPResponses(t *testing.T) {
	mux := http.NewServeMux()
	NewHandler(NewService(&failingRepository{err: errors.New("private storage failure")})).RegisterRoutes(mux)
	for _, tt := range []struct{ method, path, body string }{
		{"POST", "/api/v1/properties", `{"name":"Home","value":{"currency":"USD","amount":"1"}}`},
		{"GET", "/api/v1/properties", ""},
		{"PUT", "/api/v1/properties/" + NewID().String() + "/value", `{"currency":"USD","amount":"1"}`},
	} {
		got := serveRequest(mux, tt.method, tt.path, tt.body)
		if got.Code != 500 || got.Body.String() != "internal server error\n" {
			t.Fatalf("error response = %v", got)
		}
	}
}

func TestCanonicalPropertyID(t *testing.T) {
	id := NewID()
	for _, valid := range []string{id.String(), "00000000-0000-0000-0000-000000000000"} {
		got, err := ParseID(valid)
		if err != nil || got.String() != valid {
			t.Fatalf("ParseID(%q) = %v, %v", valid, got, err)
		}
	}
	for _, invalid := range []string{"", "bad", strings.ToUpper(id.String()), strings.ReplaceAll(id.String(), "-", ""), "urn:uuid:" + id.String(), "{" + id.String() + "}", " " + id.String()} {
		if _, err := ParseID(invalid); !errors.Is(err, ErrInvalidID) {
			t.Fatalf("accepted %q", invalid)
		}
	}
}

func TestMemoryRepositoryConcurrentReplacement(t *testing.T) {
	ctx := context.Background()
	repository := NewMemoryRepository()
	usd := parseAmount(t, "USD", "12.34")
	vnd := parseAmount(t, "VND", "5678")
	property, err := New(NewID(), " \u0085Home ", usd)
	if err != nil {
		t.Fatal(err)
	}
	if property.Name != "Home" {
		t.Fatal("name not trimmed")
	}
	if err := repository.Create(ctx, property); err != nil {
		t.Fatal(err)
	}
	if _, err := repository.ReplaceValue(ctx, NewID(), vnd); !errors.Is(err, ErrNotFound) {
		t.Fatal("missing property created by update")
	}
	list, err := repository.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	list[0].Name = "Changed copy"

	var wg sync.WaitGroup
	for i := range 50 {
		wg.Go(func() {
			value := usd
			if i%2 == 0 {
				value = vnd
			}
			updated, err := repository.ReplaceValue(ctx, property.ID, value)
			if err != nil || updated.Value != value || updated.ID != property.ID || updated.Name != property.Name {
				t.Errorf("update = %+v, %v", updated, err)
			}
			list, err := repository.List(ctx)
			if err != nil || len(list) != 1 {
				t.Errorf("list = %+v, %v", list, err)
				return
			}
			if list[0].Value != usd && list[0].Value != vnd {
				t.Error("currency and amount were not replaced atomically")
			}
		})
	}
	wg.Wait()
}

func TestServiceSortsAllPropertiesByIdentity(t *testing.T) {
	ctx := context.Background()
	repository := NewMemoryRepository()
	for _, number := range []int{3, 1, 2} {
		id, err := ParseID(fmt.Sprintf("00000000-0000-0000-0000-%012d", number))
		if err != nil {
			t.Fatal(err)
		}
		property, err := New(id, "Same name", parseAmount(t, "USD", "0"))
		if err != nil {
			t.Fatal(err)
		}
		if err := repository.Create(ctx, property); err != nil {
			t.Fatal(err)
		}
	}
	list, err := NewService(repository).ListProperties(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 3 || list[0].ID.String() >= list[1].ID.String() || list[1].ID.String() >= list[2].ID.String() {
		t.Fatalf("unordered list: %+v", list)
	}

	empty, err := NewService(NewMemoryRepository()).ListProperties(ctx)
	if err != nil || empty == nil || len(empty) != 0 {
		t.Fatalf("empty list = %#v, %v", empty, err)
	}
}
