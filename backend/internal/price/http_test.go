package price

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/mgwinsor/meridian/backend/internal/currency"

	"github.com/mgwinsor/meridian/backend/internal/instrument"
)

func request(t *testing.T, mux *http.ServeMux, method, path, body string, status int) string {
	t.Helper()
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest(method, path, strings.NewReader(body)))
	if w.Code != status {
		t.Fatalf("%s %s (%s): %d %s; want %d", method, path, body, w.Code, w.Body.String(), status)
	}
	contentType := "application/json"
	if status >= 400 {
		contentType = "text/plain; charset=utf-8"
	}
	if w.Header().Get("Content-Type") != contentType {
		t.Fatalf("Content-Type = %q; want %q", w.Header().Get("Content-Type"), contentType)
	}
	return w.Body.String()
}

func TestHTTPWorkflow(t *testing.T) {
	instruments := instrument.NewMemoryRepository()
	mux := http.NewServeMux()
	instrument.NewHandler(instrument.NewService(instruments)).RegisterRoutes(mux)
	NewHandler(NewService(instruments, NewMemoryRepository())).RegisterRoutes(mux)
	var created struct{ ID string }
	body := request(t, mux, "POST", "/api/v1/instruments", `{"kind":"stock","symbol":"AAPL","name":"Apple","quoteCurrency":"USD"}`, 201)
	if err := json.Unmarshal([]byte(body), &created); err != nil {
		t.Fatal(err)
	}
	base := "/api/v1/instruments/" + created.ID + "/prices"
	if got := request(t, mux, "GET", base, "", 200); got != "{\"observations\":[]}\n" {
		t.Fatal(got)
	}
	var want []observationResponse
	for _, tc := range []struct{ currency, input, amount, at, normalized string }{
		{"USD", " 00012.3 ", "12.30", "2026-09-14T12:00:00.123456789+08:00", "2026-09-14T04:00:00.123456789Z"},
		{"USD", "92233720368547758.07", "92233720368547758.07", "2026-09-14T03:00:00Z", "2026-09-14T03:00:00Z"},
		{"USD", "0", "0.00", "2026-09-14T03:00:00Z", "2026-09-14T03:00:00Z"},
		{"USD", "16.00", "16.00", "2026-09-14T03:00:00Z", "2026-09-14T03:00:00Z"},
	} {
		payload, err := json.Marshal(map[string]string{"amount": tc.input, "observedAt": tc.at})
		if err != nil {
			t.Fatal(err)
		}
		body := request(t, mux, "POST", base, string(payload), 201)
		var got observationResponse
		if err := json.Unmarshal([]byte(body), &got); err != nil {
			t.Fatal(err)
		}
		expected := observationResponse{InstrumentID: created.ID, Currency: tc.currency, Amount: tc.amount, ObservedAt: tc.normalized}
		if got != expected {
			t.Fatalf("response = %+v; want %+v", got, expected)
		}
		want = append(want, expected)
	}
	var list observationsResponse
	if err := json.Unmarshal([]byte(request(t, mux, "GET", base, "", 200)), &list); err != nil {
		t.Fatal(err)
	}
	if len(list.Observations) != 4 {
		t.Fatal(list)
	}
	for i, j := range []int{1, 2, 3, 0} {
		if list.Observations[i] != want[j] {
			t.Fatalf("history = %+v", list)
		}
	}
}

func TestHTTPValidation(t *testing.T) {
	repository := NewMemoryRepository()
	mux := http.NewServeMux()
	NewHandler(NewService(stubInstrumentFinder{}, repository)).RegisterRoutes(mux)
	id := instrument.NewID()
	base := "/api/v1/instruments/" + id.String() + "/prices"
	for _, tc := range []struct{ body, message string }{
		{`null`, "invalid request"},
		{`[]`, "invalid request"},
		{`{`, "invalid request"},
		{`{}`, "invalid amount"},
		{`{"amount":"1","observedAt":""}`, "invalid observation timestamp"},
		{`{"amount":1}`, "invalid request"},
		{`{"amount":null}`, "invalid amount"},
		{`{"amount":"-1"}`, "invalid amount"},
		{`{"amount":"+1"}`, "invalid amount"},
		{`{"amount":"1e2"}`, "invalid amount"},
		{`{"amount":"0.001"}`, "invalid amount"},
		{`{"amount":"92233720368547758.08"}`, "invalid amount"},
		{`{"amount":"1","observedAt":null}`, "invalid observation timestamp"},
		{`{"amount":"1","observedAt":123}`, "invalid request"},
		{`{"amount":"1","observedAt":"2026-09-14"}`, "invalid observation timestamp"},
		{`{"amount":"1","observedAt":"2026-09-14T12:00:00"}`, "invalid observation timestamp"},
		{`{"amount":"1","observedAt":"2026-02-30T12:00:00Z"}`, "invalid observation timestamp"},
		{`{"amount":"1","observedAt":"0001-01-01T00:00:00Z"}`, "invalid observation timestamp"},
		{`{"amount":"1","observedAt":"9999-12-31T23:00:00-01:00"}`, "invalid observation timestamp"},
	} {
		t.Run(tc.body, func(t *testing.T) {
			if got := request(t, mux, "POST", base, tc.body, 400); got != tc.message+"\n" {
				t.Fatalf("body = %q; want %q", got, tc.message)
			}
		})
	}
	for _, id := range []string{"bad", "ABCDEFAB-1234-4234-8234-ABCDEFABCDEF", "abcdefab123442348234abcdefabcd"} {
		for _, method := range []string{"GET", "POST"} {
			if got := request(t, mux, method, "/api/v1/instruments/"+id+"/prices", "{}", 400); got != "invalid instrument ID\n" {
				t.Fatal(got)
			}
		}
	}
	if got := request(t, mux, "GET", base, "", 200); got != "{\"observations\":[]}\n" {
		t.Fatalf("failed write persisted: %s", got)
	}
}

func TestHTTPQuoteCurrencyAndDefaultTime(t *testing.T) {
	for _, codeName := range []string{"USD", "SGD", "VND"} {
		t.Run(codeName, func(t *testing.T) {
			code, _ := currency.Parse(codeName)
			instruments := instrument.NewMemoryRepository()
			id := instrument.NewID()
			if err := instruments.Save(context.Background(), instrument.Instrument{ID: id, QuoteCurrency: code}); err != nil {
				t.Fatal(err)
			}
			mux := http.NewServeMux()
			NewHandler(NewService(instruments, NewMemoryRepository())).RegisterRoutes(mux)
			base := "/api/v1/instruments/" + id.String() + "/prices"
			before := time.Now()
			// Unknown fields remain ignored; a supplied currency cannot override the instrument.
			body := request(t, mux, "POST", base, `{"amount":"12","currency":"EUR"}`, 201)
			after := time.Now()
			var got observationResponse
			if err := json.Unmarshal([]byte(body), &got); err != nil {
				t.Fatal(err)
			}
			at, err := time.Parse(time.RFC3339Nano, got.ObservedAt)
			if err != nil || at.Before(before) || at.After(after) || !strings.HasSuffix(got.ObservedAt, "Z") {
				t.Fatalf("default timestamp = %q, %v; outside [%v, %v]", got.ObservedAt, err, before, after)
			}
			wantAmount := "12.00"
			limit, overflow := "92233720368547758.07", "92233720368547758.08"
			if codeName == "VND" {
				wantAmount = "12"
				limit, overflow = "9223372036854775807", "9223372036854775808"
				request(t, mux, "POST", base, `{"amount":"1.0"}`, 400)
			}
			if got.Currency != codeName || got.Amount != wantAmount {
				t.Fatalf("observation = %+v", got)
			}
			var history observationsResponse
			if err := json.Unmarshal([]byte(request(t, mux, "GET", base, "", 200)), &history); err != nil {
				t.Fatal(err)
			}
			if len(history.Observations) != 1 || history.Observations[0] != got {
				t.Fatalf("stored observation = %+v; want %+v", history, got)
			}
			request(t, mux, "POST", base, `{"amount":"`+limit+`"}`, 201)
			request(t, mux, "POST", base, `{"amount":"`+overflow+`"}`, 400)
		})
	}
}

func TestHTTPDependencyErrors(t *testing.T) {
	internal := errors.New("private storage details")
	for _, tc := range []struct {
		name      string
		finderErr error
		repoErr   error
		status    int
		message   string
	}{
		{"missing instrument", instrument.ErrNotFound, nil, 404, "instrument not found\n"},
		{"lookup failure", internal, nil, 500, "internal server error\n"},
		{"storage failure", nil, internal, 500, "internal server error\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mux := http.NewServeMux()
			NewHandler(NewService(stubInstrumentFinder{err: tc.finderErr}, &failingRepository{err: tc.repoErr})).RegisterRoutes(mux)
			base := "/api/v1/instruments/" + instrument.NewID().String() + "/prices"
			for _, method := range []string{"GET", "POST"} {
				got := request(t, mux, method, base, `{"amount":"1","observedAt":"2026-09-14T04:00:00Z"}`, tc.status)
				if got != tc.message {
					t.Fatalf("body = %q; want %q", got, tc.message)
				}
			}
		})
	}
}
