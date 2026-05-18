package user

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"go.uber.org/zap"
)

type Request struct {
	Email string `json:"email"`
}
type Response struct {
	Id        string    `json:"id"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

type Store interface {
	Create(ctx context.Context, email string) (User, error)
}
type Handler struct {
	logger *zap.Logger
	store  Store
}

func NewHandler(logger *zap.Logger, store Store) *Handler {
	return &Handler{
		logger: logger,
		store:  store,
	}
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req Request
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		h.logger.Error("Failed to decode request body", zap.Error(err))
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	newUser, err := h.store.Create(ctx, req.Email)
	if err != nil {
		h.logger.Error("Failed to create user", zap.Error(err))
		http.Error(w, "Failed to create user", http.StatusInternalServerError)
		return
	}
	resp := Response{
		Id:        newUser.ID.String(),
		Email:     newUser.Email,
		CreatedAt: newUser.CreatedAt.Time,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}
