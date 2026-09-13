package property

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func serveRequest(handler http.Handler, method, path, body string) *httptest.ResponseRecorder {
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(method, path, strings.NewReader(body)))
	return response
}

func setupRoutes() (*http.ServeMux, string) {
	mux := http.NewServeMux()
	NewHandler(NewService(NewMemoryRepository())).RegisterRoutes(mux)
	return mux, "/api/v1/properties"
}

func decodeProperty(t *testing.T, response *httptest.ResponseRecorder, status int) propertyResponse {
	t.Helper()
	if response.Code != status {
		t.Fatalf("status = %d, want %d: %s", response.Code, status, response.Body)
	}
	if response.Header().Get("Content-Type") != "application/json" {
		t.Fatal("missing JSON content type")
	}
	var property propertyResponse
	if err := json.Unmarshal(response.Body.Bytes(), &property); err != nil {
		t.Fatal(err)
	}
	if _, err := ParseID(property.ID); err != nil {
		t.Fatal(err)
	}
	return property
}

func TestPropertyRoutes(t *testing.T) {
	mux, path := setupRoutes()
	if got := serveRequest(mux, "GET", path, ""); got.Code != 200 || got.Body.String() != "{\"properties\":[]}\n" {
		t.Fatalf("empty list: %v", got)
	}
	response := serveRequest(mux, "POST", path, `{"name":" \u0085Home  ","value":{"currency":"SGD","amount":" 0750000.5 ","ignored":true},"id":"ignored"} {}`)
	first := decodeProperty(t, response, 201)
	if first.Name != "Home" || first.Value != (valueInput{Currency: "SGD", Amount: "750000.50"}) {
		t.Fatalf("created = %+v", first)
	}
	if response.Header().Get("Location") != "" {
		t.Fatal("unexpected Location header")
	}
	second := decodeProperty(t, serveRequest(mux, "POST", path, `{"name":"Home","value":{"currency":"USD","amount":"0"}}`), 201)
	if first.ID == second.ID {
		t.Fatal("duplicate names must create distinct identities")
	}

	valuePath := path + "/" + first.ID + "/value"
	for range 2 {
		updated := decodeProperty(t, serveRequest(mux, "PUT", valuePath, `{"currency":"VND","amount":"9223372036854775807","name":"ignored"}`), 200)
		if updated.ID != first.ID || updated.Name != first.Name || updated.Value != (valueInput{Currency: "VND", Amount: "9223372036854775807"}) {
			t.Fatalf("updated = %+v", updated)
		}
	}
	for _, input := range []struct{ currency, amount, canonical string }{
		{"USD", "92233720368547758.07", "92233720368547758.07"},
		{"SGD", "92233720368547758.07", "92233720368547758.07"},
		{"USD", "0", "0.00"},
	} {
		got := decodeProperty(t, serveRequest(mux, "PUT", valuePath, `{"currency":"`+input.currency+`","amount":"`+input.amount+`"}`), 200)
		if got.Value != (valueInput{Currency: input.currency, Amount: input.canonical}) {
			t.Fatalf("value = %+v", got.Value)
		}
	}
	var list propertiesResponse
	if err := json.Unmarshal(serveRequest(mux, "GET", path+"?ignored=true", "").Body.Bytes(), &list); err != nil {
		t.Fatal(err)
	}
	if len(list.Properties) != 2 || list.Properties[0].ID >= list.Properties[1].ID {
		t.Fatalf("list = %+v", list)
	}
	for _, property := range list.Properties {
		if property.ID == first.ID && property.Value != (valueInput{Currency: "USD", Amount: "0.00"}) {
			t.Fatal("zero was not retained")
		}
	}
}

func TestPropertyValidationAndErrorOrder(t *testing.T) {
	mux, path := setupRoutes()
	property := decodeProperty(t, serveRequest(mux, "POST", path, `{"name":"Home","value":{"currency":"USD","amount":"42"}}`), 201)
	valuePath := path + "/" + property.ID + "/value"
	type testCase struct {
		name, method, path, body, message string
		status                            int
	}
	tests := []testCase{
		{"property before JSON", "PUT", path + "/bad/value", `{`, "invalid property ID", 400},
		{"uppercase property", "PUT", path + "/" + strings.ToUpper(property.ID) + "/value", `{`, "invalid property ID", 400},
		{"compact property", "PUT", path + "/" + strings.ReplaceAll(property.ID, "-", "") + "/value", `{`, "invalid property ID", 400},
		{"name before missing value", "POST", path, `{"name":" "}`, "property name cannot be empty", 400},
		{"name before malformed value", "POST", path, `{"name":" ","value":[]}`, "property name cannot be empty", 400},
		{"missing value", "POST", path, `{"name":"Home"}`, "invalid request", 400},
		{"null value", "POST", path, `{"name":"Home","value":null}`, "invalid request", 400},
		{"nonobject value", "POST", path, `{"name":"Home","value":[]}`, "invalid request", 400},
		{"null name", "POST", path, `{"name":null,"value":{"currency":"USD","amount":"1"}}`, "property name cannot be empty", 400},
		{"number name", "POST", path, `{"name":3,"value":{"currency":"USD","amount":"1"}}`, "invalid request", 400},
		{"currency before amount and lookup", "PUT", path + "/00000000-0000-0000-0000-000000000000/value", `{"currency":"GBP","amount":"-1"}`, "unsupported currency", 400},
		{"amount before lookup", "PUT", path + "/00000000-0000-0000-0000-000000000000/value", `{"currency":"USD","amount":"-1"}`, "invalid amount", 400},
		{"missing property", "PUT", path + "/00000000-0000-0000-0000-000000000000/value", `{"currency":"USD","amount":"1"}`, "property not found", 404},
	}
	for _, body := range []string{"", `{`, `null`, `[]`, `1`, `"text"`} {
		tests = append(tests, testCase{"invalid create body " + body, "POST", path, body, "invalid request", 400})
		tests = append(tests, testCase{"invalid update body " + body, "PUT", valuePath, body, "invalid request", 400})
	}
	for _, input := range []struct{ body, message string }{
		{`{}`, "unsupported currency"},
		{`{"currency":null,"amount":"1"}`, "unsupported currency"},
		{`{"currency":"usd","amount":"1"}`, "unsupported currency"},
		{`{"currency":" USD ","amount":"1"}`, "unsupported currency"},
		{`{"currency":1,"amount":"1"}`, "invalid request"},
		{`{"currency":"USD","amount":1}`, "invalid request"},
		{`{"currency":"USD"}`, "invalid amount"},
		{`{"currency":"USD","amount":null}`, "invalid amount"},
		{`{"currency":"USD","amount":"92233720368547758.08"}`, "invalid amount"},
		{`{"currency":"SGD","amount":"92233720368547758.08"}`, "invalid amount"},
		{`{"currency":"VND","amount":"9223372036854775808"}`, "invalid amount"},
		{`{"currency":"VND","amount":"1.0"}`, "invalid amount"},
		{`{"currency":"USD","amount":"0.001"}`, "invalid amount"},
		{`{"currency":"USD","amount":"-1"}`, "invalid amount"},
		{`{"currency":"USD","amount":"1e3"}`, "invalid amount"},
		{`{"currency":"USD","amount":"1,000"}`, "invalid amount"},
		{`{"currency":"USD","amount":"+1"}`, "invalid amount"},
	} {
		tests = append(tests, testCase{"create " + input.body, "POST", path, `{"name":"Home","value":` + input.body + `}`, input.message, 400})
		tests = append(tests, testCase{"update " + input.body, "PUT", valuePath, input.body, input.message, 400})
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := serveRequest(mux, tt.method, tt.path, tt.body)
			if got.Code != tt.status || got.Body.String() != tt.message+"\n" {
				t.Fatalf("got %d %q, want %d %q", got.Code, got.Body.String(), tt.status, tt.message+"\n")
			}
			if got.Header().Get("Content-Type") != "text/plain; charset=utf-8" {
				t.Fatal("incorrect error content type")
			}
		})
	}
	var list propertiesResponse
	if err := json.Unmarshal(serveRequest(mux, "GET", path, "").Body.Bytes(), &list); err != nil {
		t.Fatal(err)
	}
	if len(list.Properties) != 1 || list.Properties[0] != property {
		t.Fatalf("failed requests changed stored properties: %+v", list)
	}
}

func TestPropertyMethodsAndRemovedNestedRoutes(t *testing.T) {
	mux, path := setupRoutes()
	for _, tt := range []struct{ path, allow string }{{path, "GET, HEAD, POST"}, {path + "/" + NewID().String() + "/value", "PUT"}} {
		got := serveRequest(mux, "DELETE", tt.path, "")
		if got.Code != 405 || got.Header().Get("Allow") != tt.allow {
			t.Fatalf("method response: %v", got)
		}
	}
	if got := serveRequest(mux, "GET", "/api/v1/accounts/00000000-0000-0000-0000-000000000000/properties", ""); got.Code != 404 {
		t.Fatalf("removed nested route status = %d", got.Code)
	}
	if got := serveRequest(mux, "HEAD", path, ""); got.Code != 200 {
		t.Fatalf("HEAD status = %d", got.Code)
	}
}
