package health_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
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
			handler := health.NewHandler(nil)
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
	handler := health.NewHandler(nil)
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
	handler := health.NewHandler(nil)
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)
	request := httptest.NewRequest(http.MethodPost, "/livez", nil)
	response := httptest.NewRecorder()

	mux.ServeHTTP(response, request)

	if response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusMethodNotAllowed)
	}
}

func TestDatabaseReadiness(t *testing.T) {
	available := false
	calls := 0
	handler := health.NewHandler(func(ctx context.Context) error {
		calls++
		if _, ok := ctx.Deadline(); !ok {
			t.Error("readiness dependency has no deadline")
		}
		if !available {
			return errors.New("private connection details")
		}
		return nil
	})
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)
	check := func(path string, status int) {
		t.Helper()
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		if response.Code != status {
			t.Fatalf("%s status = %d, want %d", path, response.Code, status)
		}
		if strings.Contains(response.Body.String(), "private") {
			t.Fatal("dependency error leaked")
		}
	}
	check("/livez", http.StatusOK)
	if calls != 0 {
		t.Fatal("liveness checked database")
	}
	check("/readyz", http.StatusServiceUnavailable)
	available = true
	check("/readyz", http.StatusOK)
	handler.BeginShutdown()
	check("/readyz", http.StatusServiceUnavailable)
	if calls != 2 {
		t.Fatalf("dependency checked %d times, want 2", calls)
	}
}
