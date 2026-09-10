package account

import (
	"encoding/json"
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

func (h Handler) Create(w http.ResponseWriter, r *http.Request) {
	type createAccountRequest struct {
		Name string `json:"name"`
	}

	type accountResponse struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	var request createAccountRequest

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	account, err := h.service.Create(r.Context(), request.Name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	_ = json.NewEncoder(w).Encode(accountResponse{
		ID:   account.ID.String(),
		Name: account.Name,
	})
}

func (h Handler) Get(w http.ResponseWriter, r *http.Request) {
}
