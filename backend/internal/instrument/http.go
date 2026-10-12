package instrument

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/mgwinsor/meridian/backend/internal/assetclass"
	"github.com/mgwinsor/meridian/backend/internal/currency"
)

type Handler struct{ service Service }

func NewHandler(service Service) Handler { return Handler{service: service} }

func (h Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("PUT /api/v1/instruments/{id}/asset-class", h.SetAssetClass)
	mux.HandleFunc("POST /api/v1/instruments", h.CreateInstrument)
	mux.HandleFunc("GET /api/v1/instruments", h.ListInstruments)
	mux.HandleFunc("GET /api/v1/instruments/{id}", h.GetInstrumentByID)
}

type createRequest struct {
	AssetClass    json.RawMessage `json:"assetClass"`
	Kind          Kind            `json:"kind"`
	Symbol        string          `json:"symbol"`
	Name          string          `json:"name"`
	QuoteCurrency string          `json:"quoteCurrency"`
}

type instrumentResponse struct {
	AssetClass    assetclass.Class `json:"assetClass"`
	ID            string           `json:"id"`
	Kind          Kind             `json:"kind"`
	Symbol        string           `json:"symbol"`
	Name          string           `json:"name"`
	QuoteCurrency string           `json:"quoteCurrency"`
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
	var requestedClass *assetclass.Class
	if len(request.AssetClass) > 0 {
		var class assetclass.Class
		if err := json.Unmarshal(request.AssetClass, &class); err != nil {
			http.Error(w, "invalid request", 400)
			return
		}
		requestedClass = &class
	}
	instrument, err := h.service.CreateInstrument(r.Context(), request.Kind, request.Symbol, request.Name, code, requestedClass)
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
	case errors.Is(err, assetclass.ErrInvalid), errors.Is(err, ErrInvalidID), errors.Is(err, ErrInvalidKind), errors.Is(err, ErrInvalidSymbol), errors.Is(err, ErrInvalidName), errors.Is(err, currency.ErrUnsupported):
		http.Error(w, err.Error(), http.StatusBadRequest)
	default:
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}

func newInstrumentResponse(instrument Instrument) instrumentResponse {
	return instrumentResponse{AssetClass: instrument.AssetClass, ID: instrument.ID.String(), Kind: instrument.Kind, Symbol: instrument.Symbol, Name: instrument.Name, QuoteCurrency: instrument.QuoteCurrency.String()}
}

func writeJSON(w http.ResponseWriter, status int, response any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(response)
}

func (h Handler) SetAssetClass(w http.ResponseWriter, r *http.Request) {
	id, err := ParseID(r.PathValue("id"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	var request *struct {
		AssetClass string `json:"assetClass"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil || request == nil {
		http.Error(w, "invalid request", 400)
		return
	}
	item, err := h.service.SetAssetClass(r.Context(), id, request.AssetClass)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, 200, newInstrumentResponse(item))
}
