package cash

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mgwinsor/meridian/backend/internal/account"
)

func TestCashRoutes(t *testing.T) {
	accountRepository := account.NewMemoryRepository()
	accountHandler := account.NewHandler(account.NewService(accountRepository))
	cashHandler := NewHandler(NewService(accountRepository, NewMemoryRepository()))
	mux := http.NewServeMux()
	accountHandler.RegisterRoutes(mux)
	cashHandler.RegisterRoutes(mux)

	createResponse := serveRequest(
		mux,
		http.MethodPost,
		"/api/v1/accounts",
		`{"name":"Primary"}`,
	)
	if createResponse.Code != http.StatusCreated {
		t.Fatalf("create account status = %d, want 201; body = %q", createResponse.Code, createResponse.Body)
	}

	var created struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(createResponse.Body).Decode(&created); err != nil {
		t.Fatalf("decode account response: %v", err)
	}

	requests := []struct {
		currency string
		amount   string
	}{
		{currency: "USD", amount: "10"},
		{currency: "SGD", amount: "20.50"},
		{currency: "USD", amount: "15.25"},
	}
	for _, request := range requests {
		response := serveRequest(
			mux,
			http.MethodPut,
			"/api/v1/accounts/"+created.ID+"/cash/"+request.currency,
			`{"amount":"`+request.amount+`"}`,
		)
		if response.Code != http.StatusOK {
			t.Fatalf("set %s status = %d, want 200; body = %q", request.currency, response.Code, response.Body)
		}
	}

	response := serveRequest(
		mux,
		http.MethodGet,
		"/api/v1/accounts/"+created.ID+"/cash",
		"",
	)
	if response.Code != http.StatusOK {
		t.Fatalf("list status = %d, want 200; body = %q", response.Code, response.Body)
	}

	var got balancesResponse
	if err := json.NewDecoder(response.Body).Decode(&got); err != nil {
		t.Fatalf("decode balances response: %v", err)
	}
	want := []balanceResponse{
		{Currency: "SGD", Amount: "20.50"},
		{Currency: "USD", Amount: "15.25"},
	}
	if len(got.Balances) != len(want) {
		t.Fatalf("balances count = %d, want %d", len(got.Balances), len(want))
	}
	for i := range want {
		if got.Balances[i] != want[i] {
			t.Errorf("balances[%d] = %+v, want %+v", i, got.Balances[i], want[i])
		}
	}
}

func TestListBalancesForExistingAccountReturnsEmptyArray(t *testing.T) {
	accountID := account.NewID()
	handler := NewHandler(NewService(existingAccountFinder(), NewMemoryRepository()))
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	response := serveRequest(
		mux,
		http.MethodGet,
		"/api/v1/accounts/"+accountID.String()+"/cash",
		"",
	)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %q", response.Code, response.Body)
	}
	if response.Body.String() != "{\"balances\":[]}\n" {
		t.Errorf("body = %q, want empty balances array", response.Body.String())
	}
}

func TestCashHTTPErrorResponses(t *testing.T) {
	validID := account.NewID().String()
	repositoryErr := errors.New("repository unavailable")

	tests := []struct {
		name       string
		method     string
		path       string
		body       string
		accounts   AccountFinder
		repository Repository
		wantStatus int
	}{
		{
			name:       "malformed account ID",
			method:     http.MethodGet,
			path:       "/api/v1/accounts/not-a-uuid/cash",
			accounts:   existingAccountFinder(),
			repository: &stubRepository{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "unsupported currency",
			method:     http.MethodPut,
			path:       "/api/v1/accounts/" + validID + "/cash/GBP",
			body:       `{"amount":"1.00"}`,
			accounts:   existingAccountFinder(),
			repository: &stubRepository{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "malformed JSON",
			method:     http.MethodPut,
			path:       "/api/v1/accounts/" + validID + "/cash/USD",
			body:       `{`,
			accounts:   existingAccountFinder(),
			repository: &stubRepository{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid amount",
			method:     http.MethodPut,
			path:       "/api/v1/accounts/" + validID + "/cash/USD",
			body:       `{"amount":"1.001"}`,
			accounts:   existingAccountFinder(),
			repository: &stubRepository{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "negative amount",
			method:     http.MethodPut,
			path:       "/api/v1/accounts/" + validID + "/cash/USD",
			body:       `{"amount":"-1.00"}`,
			accounts:   existingAccountFinder(),
			repository: &stubRepository{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing account on set",
			method:     http.MethodPut,
			path:       "/api/v1/accounts/" + validID + "/cash/USD",
			body:       `{"amount":"1.00"}`,
			accounts:   stubAccountFinder{},
			repository: &stubRepository{},
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "missing account on list",
			method:     http.MethodGet,
			path:       "/api/v1/accounts/" + validID + "/cash",
			accounts:   stubAccountFinder{},
			repository: &stubRepository{},
			wantStatus: http.StatusNotFound,
		},
		{
			name:   "account repository error",
			method: http.MethodGet,
			path:   "/api/v1/accounts/" + validID + "/cash",
			accounts: stubAccountFinder{
				findByID: func(context.Context, account.ID) (account.Account, error) {
					return account.Account{}, repositoryErr
				},
			},
			repository: &stubRepository{},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:     "cash save error",
			method:   http.MethodPut,
			path:     "/api/v1/accounts/" + validID + "/cash/USD",
			body:     `{"amount":"1.00"}`,
			accounts: existingAccountFinder(),
			repository: &stubRepository{
				save: func(context.Context, Balance) error { return repositoryErr },
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:     "cash list error",
			method:   http.MethodGet,
			path:     "/api/v1/accounts/" + validID + "/cash",
			accounts: existingAccountFinder(),
			repository: &stubRepository{
				listByAccount: func(context.Context, account.ID) ([]Balance, error) {
					return nil, repositoryErr
				},
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:       "unsupported method",
			method:     http.MethodDelete,
			path:       "/api/v1/accounts/" + validID + "/cash/USD",
			accounts:   existingAccountFinder(),
			repository: &stubRepository{},
			wantStatus: http.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewHandler(NewService(tt.accounts, tt.repository))
			mux := http.NewServeMux()
			handler.RegisterRoutes(mux)

			response := serveRequest(mux, tt.method, tt.path, tt.body)
			if response.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d; body = %q", response.Code, tt.wantStatus, response.Body)
			}
		})
	}
}

func serveRequest(handler http.Handler, method, path, body string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func TestSetBalanceResponseContentType(t *testing.T) {
	handler := NewHandler(NewService(existingAccountFinder(), NewMemoryRepository()))
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)
	id := account.NewID().String()

	response := serveRequest(
		mux,
		http.MethodPut,
		"/api/v1/accounts/"+id+"/cash/USD",
		`{"amount":"0"}`,
	)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %q", response.Code, response.Body)
	}
	if !strings.HasPrefix(response.Header().Get("Content-Type"), "application/json") {
		t.Errorf("Content-Type = %q, want application/json", response.Header().Get("Content-Type"))
	}
}
