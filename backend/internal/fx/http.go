package fx

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/mgwinsor/meridian/backend/internal/currency"
)

type Handler struct{ service Service }

func NewHandler(service Service) Handler { return Handler{service: service} }

func (h Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/fx-rates/{currency}", h.RecordObservation)
	mux.HandleFunc("GET /api/v1/fx-rates/{currency}", h.ListObservations)
}

type recordObservationRequest struct {
	Rate       string          `json:"rate"`
	ObservedAt json.RawMessage `json:"observedAt,omitempty"`
}

type observationResponse struct {
	Currency   string `json:"currency"`
	Rate       string `json:"rate"`
	ObservedAt string `json:"observedAt"`
}

type observationsResponse struct {
	Observations []observationResponse `json:"observations"`
}

func (h Handler) RecordObservation(w http.ResponseWriter, r *http.Request) {
	code, err := currency.Parse(r.PathValue("currency"))
	if err != nil || code.String() != r.PathValue("currency") || code.String() == "USD" {
		writeServiceError(w, currency.ErrUnsupported)
		return
	}
	var request *recordObservationRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil || request == nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	observedAt := time.Now()
	if len(request.ObservedAt) != 0 {
		var value string
		if err := json.Unmarshal(request.ObservedAt, &value); err != nil {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}
		observedAt, err = time.Parse(time.RFC3339Nano, value)
		if err != nil {
			writeServiceError(w, ErrInvalidObservedAt)
			return
		}
	}
	observation, err := h.service.RecordObservation(r.Context(), code, request.Rate, observedAt)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, newObservationResponse(observation))
}

func (h Handler) ListObservations(w http.ResponseWriter, r *http.Request) {
	code, err := currency.Parse(r.PathValue("currency"))
	if err != nil || code.String() != r.PathValue("currency") || code.String() == "USD" {
		writeServiceError(w, currency.ErrUnsupported)
		return
	}
	observations, err := h.service.ListObservations(r.Context(), code)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	response := observationsResponse{Observations: make([]observationResponse, 0, len(observations))}
	for _, observation := range observations {
		response.Observations = append(response.Observations, newObservationResponse(observation))
	}
	writeJSON(w, http.StatusOK, response)
}

func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, currency.ErrUnsupported), errors.Is(err, ErrInvalidRate), errors.Is(err, ErrInvalidObservedAt):
		http.Error(w, err.Error(), http.StatusBadRequest)
	default:
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}

func newObservationResponse(observation Observation) observationResponse {
	return observationResponse{
		Currency:   observation.Currency.String(),
		Rate:       observation.Rate.String(),
		ObservedAt: observation.ObservedAt.Format(time.RFC3339Nano),
	}
}

func writeJSON(w http.ResponseWriter, status int, response any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(response)
}
