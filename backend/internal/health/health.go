package health

import (
	"net/http"
	"sync/atomic"
)

type Handler struct {
	isShuttingDown atomic.Bool
}

func NewHandler() *Handler {
	return &Handler{}
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

func (h *Handler) readiness(w http.ResponseWriter, _ *http.Request) {
	w.Header().Add("Content-Type", "text/plain; charset=utf-8")

	if h.isShuttingDown.Load() {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte("shutting down"))
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(http.StatusText(http.StatusOK)))
}
