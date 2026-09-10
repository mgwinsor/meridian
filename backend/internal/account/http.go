package account

import (
	"encoding/json"
	"errors"
	"net/http"
)

type Handler struct {
	service Service
}

func NewHandler(service Service) Handler {
	return Handler{
		service: service,
	}
}

func (h Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/accounts", h.CreateAccount)
	mux.HandleFunc("GET /api/v1/accounts/{id}", h.GetAccountByID)
}

type createAccountRequest struct {
	Name string `json:"name"`
}

type accountResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func (h Handler) CreateAccount(w http.ResponseWriter, r *http.Request) {
	var request createAccountRequest

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	account, err := h.service.CreateAccount(r.Context(), request.Name)
	if err != nil {
		if errors.Is(err, ErrInvalidName) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	_ = json.NewEncoder(w).Encode(accountResponse{
		ID:   account.ID.String(),
		Name: account.Name,
	})
}

func (h Handler) GetAccountByID(w http.ResponseWriter, r *http.Request) {
	id, err := ParseID(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid account ID", http.StatusBadRequest)
		return
	}

	account, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			http.Error(w, "account not found", http.StatusNotFound)
			return
		}

		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	_ = json.NewEncoder(w).Encode(accountResponse{
		ID:   account.ID.String(),
		Name: account.Name,
	})
}
