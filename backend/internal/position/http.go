package position

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/mgwinsor/meridian/backend/internal/account"
	"github.com/mgwinsor/meridian/backend/internal/instrument"
)

type Handler struct {
	service Service
}

func NewHandler(service Service) Handler {
	return Handler{service: service}
}

func (h Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("PUT /api/v1/accounts/{id}/positions/{instrumentID}", h.SetPosition)
	mux.HandleFunc("GET /api/v1/accounts/{id}/positions", h.ListPositions)
}

type setPositionRequest struct {
	Quantity string `json:"quantity"`
}

type positionResponse struct {
	AccountID    string `json:"accountId"`
	InstrumentID string `json:"instrumentId"`
	Quantity     string `json:"quantity"`
}

type positionsResponse struct {
	Positions []positionResponse `json:"positions"`
}

func (h Handler) SetPosition(w http.ResponseWriter, r *http.Request) {
	accountID, ok := parseAccountID(w, r)
	if !ok {
		return
	}

	instrumentID, err := instrument.ParseID(r.PathValue("instrumentID"))
	if err != nil {
		http.Error(w, "invalid instrument ID", http.StatusBadRequest)
		return
	}

	var request setPositionRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	quantity, err := ParseQuantity(request.Quantity)
	if err != nil {
		http.Error(w, "invalid quantity", http.StatusBadRequest)
		return
	}

	position, err := h.service.SetPosition(r.Context(), accountID, instrumentID, quantity)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, newPositionResponse(position))
}

func (h Handler) ListPositions(w http.ResponseWriter, r *http.Request) {
	accountID, ok := parseAccountID(w, r)
	if !ok {
		return
	}

	positions, err := h.service.ListPositions(r.Context(), accountID)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	response := positionsResponse{Positions: make([]positionResponse, 0, len(positions))}
	for _, position := range positions {
		response.Positions = append(response.Positions, newPositionResponse(position))
	}

	writeJSON(w, http.StatusOK, response)
}

func parseAccountID(w http.ResponseWriter, r *http.Request) (account.ID, bool) {
	id, err := account.ParseID(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid account ID", http.StatusBadRequest)
		return account.ID{}, false
	}

	return id, true
}

func writeServiceError(w http.ResponseWriter, err error) {
	if errors.Is(err, account.ErrNotFound) {
		http.Error(w, "account not found", http.StatusNotFound)
		return
	}

	if errors.Is(err, instrument.ErrNotFound) {
		http.Error(w, "instrument not found", http.StatusNotFound)
		return
	}

	http.Error(w, "internal server error", http.StatusInternalServerError)
}

func newPositionResponse(position Position) positionResponse {
	return positionResponse{
		AccountID:    position.AccountID.String(),
		InstrumentID: position.InstrumentID.String(),
		Quantity:     position.Quantity.String(),
	}
}

func writeJSON(w http.ResponseWriter, status int, response any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(response)
}
