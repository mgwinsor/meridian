package cash

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/mgwinsor/meridian/backend/internal/account"
	"github.com/mgwinsor/meridian/backend/internal/currency"
	"github.com/mgwinsor/meridian/backend/internal/money"
)

type Handler struct {
	service Service
}

func NewHandler(service Service) Handler {
	return Handler{service: service}
}

func (h Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("PUT /api/v1/accounts/{id}/cash/{currency}", h.SetBalance)
	mux.HandleFunc("GET /api/v1/accounts/{id}/cash", h.ListBalances)
}

type setBalanceRequest struct {
	Amount string `json:"amount"`
}

type balanceResponse struct {
	Currency string `json:"currency"`
	Amount   string `json:"amount"`
}

type balancesResponse struct {
	Balances []balanceResponse `json:"balances"`
}

func (h Handler) SetBalance(w http.ResponseWriter, r *http.Request) {
	accountID, ok := parseAccountID(w, r)
	if !ok {
		return
	}

	code, err := currency.Parse(r.PathValue("currency"))
	if err != nil {
		http.Error(w, "unsupported currency", http.StatusBadRequest)
		return
	}

	var request setBalanceRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	amount, err := money.Parse(code, request.Amount)
	if err != nil {
		http.Error(w, "invalid amount", http.StatusBadRequest)
		return
	}

	balance, err := h.service.SetBalance(r.Context(), accountID, amount)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, newBalanceResponse(balance))
}

func (h Handler) ListBalances(w http.ResponseWriter, r *http.Request) {
	accountID, ok := parseAccountID(w, r)
	if !ok {
		return
	}

	balances, err := h.service.ListBalances(r.Context(), accountID)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	response := balancesResponse{Balances: make([]balanceResponse, 0, len(balances))}
	for _, balance := range balances {
		response.Balances = append(response.Balances, newBalanceResponse(balance))
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

	http.Error(w, "internal server error", http.StatusInternalServerError)
}

func newBalanceResponse(balance Balance) balanceResponse {
	return balanceResponse{
		Currency: balance.Amount.Currency.String(),
		Amount:   balance.Amount.String(),
	}
}

func writeJSON(w http.ResponseWriter, status int, response any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(response)
}
