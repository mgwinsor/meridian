package property

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/mgwinsor/meridian/backend/internal/currency"
	"github.com/mgwinsor/meridian/backend/internal/money"
)

type Handler struct{ service Service }

func NewHandler(service Service) Handler { return Handler{service: service} }

func (h Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/properties", h.CreateProperty)
	mux.HandleFunc("GET /api/v1/properties", h.ListProperties)
	mux.HandleFunc("PUT /api/v1/properties/{propertyId}/value", h.SetValue)
}

type valueInput struct {
	Currency string `json:"currency"`
	Amount   string `json:"amount"`
}

type createRequest struct {
	Name  string          `json:"name"`
	Value json.RawMessage `json:"value"`
}

type propertyResponse struct {
	ID    string     `json:"id"`
	Name  string     `json:"name"`
	Value valueInput `json:"value"`
}

type propertiesResponse struct {
	Properties []propertyResponse `json:"properties"`
}

func (h Handler) CreateProperty(w http.ResponseWriter, r *http.Request) {
	var request *createRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil || request == nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(request.Name) == "" {
		http.Error(w, ErrInvalidName.Error(), http.StatusBadRequest)
		return
	}
	var input *valueInput
	if len(request.Value) == 0 || json.Unmarshal(request.Value, &input) != nil || input == nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	value, ok := parseValue(w, *input)
	if !ok {
		return
	}
	property, err := h.service.CreateProperty(r.Context(), request.Name, value)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, newPropertyResponse(property))
}

func (h Handler) ListProperties(w http.ResponseWriter, r *http.Request) {
	properties, err := h.service.ListProperties(r.Context())
	if err != nil {
		writeServiceError(w, err)
		return
	}
	response := propertiesResponse{Properties: make([]propertyResponse, 0, len(properties))}
	for _, property := range properties {
		response.Properties = append(response.Properties, newPropertyResponse(property))
	}
	writeJSON(w, http.StatusOK, response)
}

func (h Handler) SetValue(w http.ResponseWriter, r *http.Request) {
	id, err := ParseID(r.PathValue("propertyId"))
	if err != nil {
		http.Error(w, "invalid property ID", http.StatusBadRequest)
		return
	}
	var request *valueInput
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil || request == nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	value, ok := parseValue(w, *request)
	if !ok {
		return
	}
	property, err := h.service.SetValue(r.Context(), id, value)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, newPropertyResponse(property))
}

func parseValue(w http.ResponseWriter, input valueInput) (money.Amount, bool) {
	code, err := currency.Parse(input.Currency)
	if err != nil || code.String() != input.Currency {
		http.Error(w, "unsupported currency", http.StatusBadRequest)
		return money.Amount{}, false
	}
	value, err := money.Parse(code, input.Amount)
	if err != nil {
		http.Error(w, "invalid amount", http.StatusBadRequest)
		return money.Amount{}, false
	}
	return value, true
}

func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		http.Error(w, "property not found", http.StatusNotFound)
	case errors.Is(err, ErrInvalidName):
		http.Error(w, ErrInvalidName.Error(), http.StatusBadRequest)
	default:
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}

func newPropertyResponse(property Property) propertyResponse {
	return propertyResponse{ID: property.ID.String(), Name: property.Name, Value: valueInput{Currency: property.Value.Currency.String(), Amount: property.Value.String()}}
}

func writeJSON(w http.ResponseWriter, status int, response any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(response)
}
