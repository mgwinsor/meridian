package health

import (
	"context"
	"net/http"
	"sync/atomic"
	"time"
)

type Handler struct {
	isShuttingDown  atomic.Bool
	checkDependency func(context.Context) error
}

func NewHandler(checkDependency func(context.Context) error) *Handler {
	return &Handler{checkDependency: checkDependency}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /livez", h.liveness)
	mux.HandleFunc("GET /readyz", h.readiness)
}

func (h *Handler) BeginShutdown() {
	h.isShuttingDown.Store(true)
}

func (h *Handler) liveness(w http.ResponseWriter, _ *http.Request) {
	w.Header().Add("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(http.StatusText(http.StatusOK)))
}

func (h *Handler) readiness(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/plain; charset=utf-8")

	if h.isShuttingDown.Load() {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte("shutting down"))
		return
	}

	if h.checkDependency != nil {
		ctx, cancel := context.WithTimeout(r.Context(), time.Second)
		defer cancel()
		if err := h.checkDependency(ctx); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte("database unavailable"))
			return
		}
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(http.StatusText(http.StatusOK)))
}
