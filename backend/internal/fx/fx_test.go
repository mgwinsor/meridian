package fx

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/mgwinsor/meridian/backend/internal/currency"
)

func TestRate(t *testing.T) {
	for _, s := range []string{
		"0", "-1", "+1", "1e3", "NaN", "1,000", "1.", ".1",
		"100000000000000000000", "0.0000000000000000001",
	} {
		if _, err := ParseRate(s); err == nil {
			t.Errorf("accepted %q", s)
		}
	}
	for _, s := range []string{
		"0.000000000000000001", "99999999999999999999.999999999999999999",
		" 0002.5000000000000000000 ",
	} {
		if _, err := ParseRate(s); err != nil {
			t.Errorf("rejected %q: %v", s, err)
		}
	}
}

func TestHistoryAndLatest(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryRepository()
	code, _ := currency.Parse("SGD")
	service := NewService(repo)
	at := time.Date(2026, 1, 1, 0, 0, 0, 123456789, time.FixedZone("offset", 3600))
	for _, item := range []struct {
		value string
		at    time.Time
	}{{"2", at}, {"9", at.Add(time.Hour)}, {"1", at.Add(-time.Hour)}, {"3", at}} {
		if _, err := service.RecordObservation(ctx, code, item.value, item.at); err != nil {
			t.Fatal(err)
		}
	}
	latest, err := repo.Latest(ctx, at)
	if err != nil || len(latest) != 1 || latest[0].Rate.String() != "3" ||
		latest[0].ObservedAt.Location() != time.UTC || latest[0].ObservedAt.Nanosecond() != 123456000 {
		t.Fatalf("%+v %v", latest, err)
	}
	history, _ := service.ListObservations(ctx, code)
	if len(history) != 4 || history[1].Rate.String() != "2" || history[2].Rate.String() != "3" {
		t.Fatal(history)
	}
	latest, _ = repo.Latest(ctx, at.Add(-2*time.Hour))
	if latest == nil || len(latest) != 0 {
		t.Fatal(latest)
	}
}

func TestHTTP(t *testing.T) {
	mux := http.NewServeMux()
	NewHandler(NewService(NewMemoryRepository())).RegisterRoutes(mux)
	for _, tc := range []struct {
		method, path, body string
		status             int
	}{
		{"GET", "SGD", "", 200}, {"POST", "SGD", `{"rate":"2","observedAt":"2026-01-01T08:00:00.123456789+08:00"}`, 201},
		{"POST", "VND", `{"rate":"0.123456789123456789"}`, 201},
		{"POST", "USD", `{"rate":"1"}`, 400}, {"GET", "usd", "", 400}, {"GET", "EUR", "", 400},
		{"POST", "SGD", `{"rate":"0"}`, 400}, {"POST", "SGD", `{"rate":"1","observedAt":null}`, 400},
		{"POST", "SGD", `{"rate":"1","observedAt":"2026-01-01T08:00:00"}`, 400}, {"POST", "SGD", `null`, 400},
	} {
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, httptest.NewRequest(tc.method, "/api/v1/fx-rates/"+tc.path, strings.NewReader(tc.body)))
		if w.Code != tc.status {
			t.Fatalf("%+v: %d %s", tc, w.Code, w.Body.String())
		}
	}
}
