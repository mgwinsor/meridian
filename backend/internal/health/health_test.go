package health_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mgwinsor/meridian/backend/internal/health"
)

func TestHealthRoutes(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{name: "liveness", path: "/livez"},
		{name: "readiness", path: "/readyz"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := health.NewHandler()
			mux := http.NewServeMux()
			handler.RegisterRoutes(mux)
			request := httptest.NewRequest(http.MethodGet, tt.path, nil)
			response := httptest.NewRecorder()

			mux.ServeHTTP(response, request)

			if response.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
			}
		})
	}
}

func TestReadinessDuringShutdown(t *testing.T) {
	handler := health.NewHandler()
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)
	handler.BeginShutdown()
	request := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	response := httptest.NewRecorder()

	mux.ServeHTTP(response, request)

	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusServiceUnavailable)
	}
}

func TestHealthRoutesRejectUnsupportedMethods(t *testing.T) {
	handler := health.NewHandler()
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)
	request := httptest.NewRequest(http.MethodPost, "/livez", nil)
	response := httptest.NewRecorder()

	mux.ServeHTTP(response, request)

	if response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusMethodNotAllowed)
	}
}
