package instrument

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func request(mux *http.ServeMux, method, path, body string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest(method, path, strings.NewReader(body)))
	return w
}

const validBody = `{"kind":"stock","symbol":" AAPL ","name":" Apple ","quoteCurrency":"USD"}`

func TestHTTPLifecycle(t *testing.T) {
	mux := http.NewServeMux()
	NewHandler(NewService(NewMemoryRepository())).RegisterRoutes(mux)
	w := request(mux, "GET", "/api/v1/instruments", "")
	if w.Code != 200 || w.Body.String() != "{\"instruments\":[]}\n" {
		t.Fatalf("empty: %d %s", w.Code, w.Body.String())
	}
	w = request(mux, "POST", "/api/v1/instruments", validBody)
	if w.Code != 201 || w.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("create: %d %s", w.Code, w.Body.String())
	}
	var created instrumentResponse
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if _, err := ParseID(created.ID); err != nil {
		t.Fatal(err)
	}
	if created.Kind != KindStock || created.Symbol != "AAPL" || created.Name != "Apple" || created.QuoteCurrency != "USD" {
		t.Fatalf("response: %+v", created)
	}
	var fields map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &fields); err != nil || len(fields) != 5 {
		t.Fatalf("unexpected fields: %v, %v", fields, err)
	}
	get := request(mux, "GET", "/api/v1/instruments/"+created.ID, "")
	if get.Code != 200 || get.Body.String() != w.Body.String() {
		t.Fatalf("get: %d %s", get.Code, get.Body.String())
	}
	for _, code := range []string{"SGD", "VND"} {
		w = request(mux, "POST", "/api/v1/instruments", strings.ReplaceAll(validBody, "USD", code))
		if w.Code != 201 {
			t.Fatalf("currency %s: %d %s", code, w.Code, w.Body.String())
		}
	}
	w = request(mux, "GET", "/api/v1/instruments", "")
	var list instrumentsResponse
	if err := json.Unmarshal(w.Body.Bytes(), &list); err != nil || len(list.Instruments) != 3 {
		t.Fatalf("list: %s, %v", w.Body.String(), err)
	}
	for i := 1; i < len(list.Instruments); i++ {
		if list.Instruments[i-1].ID >= list.Instruments[i].ID {
			t.Fatal("unordered list")
		}
	}
}

func TestHTTPErrors(t *testing.T) {
	mux := http.NewServeMux()
	NewHandler(NewService(NewMemoryRepository())).RegisterRoutes(mux)
	for _, tc := range []struct{ body, want string }{
		{"", "invalid request"}, {"null", "invalid request"}, {"[]", "invalid request"}, {"{", "invalid request"},
		{`{"kind":5}`, "invalid request"},
		{`{}`, "unsupported currency"},
		{strings.ReplaceAll(validBody, "USD", "EUR"), "unsupported currency"},
		{strings.ReplaceAll(validBody, "USD", "usd"), "unsupported currency"},
		{strings.ReplaceAll(validBody, "USD", " USD "), "unsupported currency"},
		{strings.ReplaceAll(validBody, `"USD"`, "null"), "unsupported currency"},
		{strings.ReplaceAll(validBody, "stock", "option"), "unsupported instrument kind"},
		{strings.ReplaceAll(validBody, " AAPL ", " "), "instrument symbol cannot be empty"},
		{strings.ReplaceAll(validBody, " Apple ", " "), "instrument name cannot be empty"},
	} {
		w := request(mux, "POST", "/api/v1/instruments", tc.body)
		if w.Code != 400 || w.Body.String() != tc.want+"\n" {
			t.Errorf("body %s: %d %s", tc.body, w.Code, w.Body.String())
		}
	}
	for _, tc := range []struct {
		method, path string
		status       int
		body         string
	}{
		{"GET", "/api/v1/instruments/bad-id", 400, "invalid instrument ID\n"},
		{"GET", "/api/v1/instruments/123E4567-E89B-42D3-A456-426614174000", 400, "invalid instrument ID\n"},
		{"GET", "/api/v1/instruments/" + NewID().String(), 404, "instrument not found\n"},
		{"PUT", "/api/v1/instruments", 405, "Method Not Allowed\n"},
	} {
		w := request(mux, tc.method, tc.path, "")
		if w.Code != tc.status || w.Body.String() != tc.body {
			t.Errorf("%s %s: %d %s", tc.method, tc.path, w.Code, w.Body.String())
		}
	}
	w := request(mux, "GET", "/api/v1/instruments", "")
	if w.Body.String() != "{\"instruments\":[]}\n" {
		t.Fatal("validation failures persisted data")
	}
}

func TestHTTPRepositoryFailures(t *testing.T) {
	mux := http.NewServeMux()
	NewHandler(NewService(failingRepository{errors.New("secret storage details")})).RegisterRoutes(mux)
	for _, tc := range []struct{ method, path, body string }{
		{"POST", "/api/v1/instruments", validBody},
		{"GET", "/api/v1/instruments", ""},
		{"GET", "/api/v1/instruments/" + NewID().String(), ""},
	} {
		w := request(mux, tc.method, tc.path, tc.body)
		if w.Code != 500 || w.Body.String() != "internal server error\n" {
			t.Fatalf("failure leaked: %d %s", w.Code, w.Body.String())
		}
	}
}
