package overview

import (
	"encoding/json"
	"net/http"

	"github.com/mgwinsor/meridian/backend/internal/currency"
)

type Handler struct{ service Service }

func NewHandler(service Service) Handler            { return Handler{service} }
func (h Handler) RegisterRoutes(mux *http.ServeMux) { mux.HandleFunc("GET /api/v1/overview", h.Get) }
func (h Handler) Get(w http.ResponseWriter, r *http.Request) {
	value := "USD"
	if r.URL.Query().Has("currency") {
		value = r.URL.Query().Get("currency")
	}
	code, err := currency.Parse(value)
	if err != nil || code.String() != value {
		http.Error(w, currency.ErrUnsupported.Error(), 400)
		return
	}
	report, err := h.service.Generate(r.Context(), code)
	if err != nil {
		http.Error(w, "internal server error", 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(report)
}
