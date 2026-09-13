package instrument

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/mgwinsor/meridian/backend/internal/currency"
)

type Handler struct{ service Service }

func NewHandler(service Service) Handler { return Handler{service: service} }

func (h Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/instruments", h.CreateInstrument)
	mux.HandleFunc("GET /api/v1/instruments", h.ListInstruments)
	mux.HandleFunc("GET /api/v1/instruments/{id}", h.GetInstrumentByID)
}

type createRequest struct {
	Kind          Kind   `json:"kind"`
	Symbol        string `json:"symbol"`
	Name          string `json:"name"`
	QuoteCurrency string `json:"quoteCurrency"`
}

type instrumentResponse struct {
	ID            string `json:"id"`
	Kind          Kind   `json:"kind"`
	Symbol        string `json:"symbol"`
	Name          string `json:"name"`
	QuoteCurrency string `json:"quoteCurrency"`
}

type instrumentsResponse struct {
	Instruments []instrumentResponse `json:"instruments"`
}

func (h Handler) CreateInstrument(w http.ResponseWriter, r *http.Request) {
	var request *createRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil || request == nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	code, err := currency.Parse(request.QuoteCurrency)
	if err != nil || code.String() != request.QuoteCurrency {
		writeServiceError(w, currency.ErrUnsupported)
		return
	}
	instrument, err := h.service.CreateInstrument(r.Context(), request.Kind, request.Symbol, request.Name, code)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, newInstrumentResponse(instrument))
}

func (h Handler) ListInstruments(w http.ResponseWriter, r *http.Request) {
	instruments, err := h.service.ListInstruments(r.Context())
	if err != nil {
		writeServiceError(w, err)
		return
	}
	response := instrumentsResponse{Instruments: make([]instrumentResponse, 0, len(instruments))}
	for _, instrument := range instruments {
		response.Instruments = append(response.Instruments, newInstrumentResponse(instrument))
	}
	writeJSON(w, http.StatusOK, response)
}

func (h Handler) GetInstrumentByID(w http.ResponseWriter, r *http.Request) {
	id, err := ParseID(r.PathValue("id"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	instrument, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, newInstrumentResponse(instrument))
}

func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		http.Error(w, ErrNotFound.Error(), http.StatusNotFound)
	case errors.Is(err, ErrInvalidID), errors.Is(err, ErrInvalidKind), errors.Is(err, ErrInvalidSymbol), errors.Is(err, ErrInvalidName), errors.Is(err, currency.ErrUnsupported):
		http.Error(w, err.Error(), http.StatusBadRequest)
	default:
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}

func newInstrumentResponse(instrument Instrument) instrumentResponse {
	return instrumentResponse{ID: instrument.ID.String(), Kind: instrument.Kind, Symbol: instrument.Symbol, Name: instrument.Name, QuoteCurrency: instrument.QuoteCurrency.String()}
}

func writeJSON(w http.ResponseWriter, status int, response any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(response)
}
