package account

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAccountRoutes(t *testing.T) {
	repository := NewMemoryRepository()
	handler := NewHandler(NewService(repository))
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	createRequest := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/accounts",
		bytes.NewBufferString(`{"name":"Primary"}`),
	)
	createResponse := httptest.NewRecorder()
	mux.ServeHTTP(createResponse, createRequest)
	if createResponse.Code != http.StatusCreated {
		t.Fatalf(
			"POST status = %d, want %d; body = %q",
			createResponse.Code,
			http.StatusCreated,
			createResponse.Body.String(),
		)
	}

	var created accountResponse
	if err := json.NewDecoder(createResponse.Body).Decode(&created); err != nil {
		t.Fatalf("decode POST response: %v", err)
	}

	createdID := created.ID
	getRequest := httptest.NewRequest(http.MethodGet, "/api/v1/accounts/"+createdID, nil)
	getResponse := httptest.NewRecorder()
	mux.ServeHTTP(getResponse, getRequest)
	if getResponse.Code != http.StatusOK {
		t.Fatalf(
			"GET status = %d, want %d; body = %q",
			getResponse.Code,
			http.StatusOK,
			getResponse.Body.String(),
		)
	}
	if !strings.Contains(getResponse.Body.String(), `"name":"Primary"`) {
		t.Errorf("GET body = %q, want created account", getResponse.Body.String())
	}

	listRequest := httptest.NewRequest(http.MethodGet, "/api/v1/accounts", nil)
	listResponse := httptest.NewRecorder()
	mux.ServeHTTP(listResponse, listRequest)
	if listResponse.Code != http.StatusOK {
		t.Fatalf(
			"list GET status = %d, want %d; body = %q",
			listResponse.Code,
			http.StatusOK,
			listResponse.Body.String(),
		)
	}

	var listed accountsResponse
	if err := json.NewDecoder(listResponse.Body).Decode(&listed); err != nil {
		t.Fatalf("decode list GET response: %v", err)
	}
	if len(listed.Accounts) != 1 || listed.Accounts[0] != created {
		t.Errorf("list GET accounts = %#v, want [%#v]", listed.Accounts, created)
	}
}

func TestListAccountsReturnsEmptyArray(t *testing.T) {
	handler := NewHandler(NewService(NewMemoryRepository()))
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/accounts", nil)
	response := httptest.NewRecorder()

	mux.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %q", response.Code, http.StatusOK, response.Body.String())
	}
	if response.Header().Get("Content-Type") != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", response.Header().Get("Content-Type"))
	}
	if response.Body.String() != "{\"accounts\":[]}\n" {
		t.Errorf("body = %q, want empty accounts array", response.Body.String())
	}
}

func TestAccountCollectionUnsupportedMethodIncludesAllowedMethods(t *testing.T) {
	handler := NewHandler(NewService(NewMemoryRepository()))
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)
	request := httptest.NewRequest(http.MethodDelete, "/api/v1/accounts", nil)
	response := httptest.NewRecorder()

	mux.ServeHTTP(response, request)

	if response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d; body = %q", response.Code, http.StatusMethodNotAllowed, response.Body.String())
	}
	if response.Header().Get("Allow") != "GET, HEAD, POST" {
		t.Errorf("Allow = %q, want %q", response.Header().Get("Allow"), "GET, HEAD, POST")
	}
}

func TestAccountHTTPErrorResponses(t *testing.T) {
	repositoryErr := errors.New("repository unavailable")

	tests := []struct {
		name       string
		method     string
		path       string
		body       string
		repository Repository
		wantStatus int
	}{
		{
			name:       "malformed JSON",
			method:     http.MethodPost,
			path:       "/api/v1/accounts",
			body:       `{`,
			repository: &stubRepository{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "blank name",
			method:     http.MethodPost,
			path:       "/api/v1/accounts",
			body:       `{"name":" "}`,
			repository: &stubRepository{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:   "create repository error",
			method: http.MethodPost,
			path:   "/api/v1/accounts",
			body:   `{"name":"Primary"}`,
			repository: &stubRepository{
				save: func(context.Context, Account) error {
					return repositoryErr
				},
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:       "malformed ID",
			method:     http.MethodGet,
			path:       "/api/v1/accounts/not-a-uuid",
			repository: &stubRepository{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing account",
			method:     http.MethodGet,
			path:       "/api/v1/accounts/00000000-0000-0000-0000-000000000001",
			repository: &stubRepository{},
			wantStatus: http.StatusNotFound,
		},
		{
			name:   "get repository error",
			method: http.MethodGet,
			path:   "/api/v1/accounts/00000000-0000-0000-0000-000000000001",
			repository: &stubRepository{
				findByID: func(context.Context, ID) (Account, error) {
					return Account{}, repositoryErr
				},
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:   "list repository error",
			method: http.MethodGet,
			path:   "/api/v1/accounts",
			repository: &stubRepository{
				list: func(context.Context) ([]Account, error) {
					return nil, repositoryErr
				},
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:       "unsupported method",
			method:     http.MethodDelete,
			path:       "/api/v1/accounts/00000000-0000-0000-0000-000000000001",
			repository: &stubRepository{},
			wantStatus: http.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewHandler(NewService(tt.repository))
			mux := http.NewServeMux()
			handler.RegisterRoutes(mux)
			request := httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body))
			response := httptest.NewRecorder()

			mux.ServeHTTP(response, request)

			if response.Code != tt.wantStatus {
				t.Errorf(
					"status = %d, want %d; body = %q",
					response.Code,
					tt.wantStatus,
					response.Body.String(),
				)
			}
		})
	}
}
