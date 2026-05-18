package form

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"awesomeProject/internal/jwt"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type Request struct {
	Title       string `json:"title" validate:"required"`
	Description string `json:"description" validate:"required"`
}

type Response struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"createdAt"`
	IsBookmark  bool      `json:"is_bookmark"`
}

// interface Store宣告method，由struct來實現

//go:generate mockery --name=Store
type Store interface {
	Create(ctx context.Context, name, description string, author_id uuid.UUID) (Form, error)
	List(ctx context.Context, userID uuid.UUID) ([]ListRow, error)
	Update(ctx context.Context, id uuid.UUID, title, description string) (Form, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type Handler struct {
	logger    *zap.Logger
	validator *validator.Validate
	store     Store //掛載interface
}

func NewHandler(logger *zap.Logger, validator *validator.Validate, store Store) *Handler {
	return &Handler{
		logger:    logger,
		validator: validator,
		store:     store,
	}
}

// 實現interface說的method: Create
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req Request
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		h.logger.Error("Failed to decode request body", zap.Error(err))
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	// valid the request
	err = h.validator.Struct(req)
	if err != nil {
		h.logger.Error("Validation failed", zap.Error(err))
		http.Error(w, "Validation failed", http.StatusBadRequest)
		return
	}

	// 從 context 取出 middleware 存入的 user ID
	authorID, ok := ctx.Value(jwt.UserContextKey).(uuid.UUID)
	if !ok {
		h.logger.Error("Failed to get user ID from context")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	newForm, err := h.store.Create(ctx, req.Title, req.Description, authorID)
	if err != nil {
		h.logger.Error("Failed to create form", zap.Error(err))
		http.Error(w, "Failed to create form", http.StatusInternalServerError)
		return
	}

	resp := Response{
		ID:          newForm.ID.String(),
		Title:       newForm.Title,
		Description: newForm.Description.String,
		CreatedAt:   newForm.CreatedAt.Time,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, ok := ctx.Value(jwt.UserContextKey).(uuid.UUID)
	if !ok {
		h.logger.Error("Failed to get user ID from context")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	forms, err := h.store.List(ctx, userID)
	if err != nil {
		h.logger.Error("Failed to list forms", zap.Error(err))
		http.Error(w, "Failed to list forms", http.StatusInternalServerError)
		return
	}

	var resp []Response
	for _, f := range forms {
		resp = append(resp, Response{
			ID:          f.ID.String(),
			Title:       f.Title,
			Description: f.Description.String,
			CreatedAt:   f.CreatedAt.Time,
			IsBookmark:  f.IsBookmark,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	idStr := r.PathValue("id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		h.logger.Error("Invalid form ID", zap.String("id", idStr), zap.Error(err))
		http.Error(w, "Invalid form ID", http.StatusBadRequest)
		return
	}

	var req Request
	err = json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		h.logger.Error("Failed to decode request body", zap.Error(err))
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	updatedForm, err := h.store.Update(ctx, id, req.Title, req.Description)
	if err != nil {
		h.logger.Error("Failed to update form", zap.Error(err))
		http.Error(w, "Failed to update form", http.StatusInternalServerError)
		return
	}

	resp := Response{
		ID:          updatedForm.ID.String(),
		Title:       updatedForm.Title,
		Description: updatedForm.Description.String,
		CreatedAt:   updatedForm.CreatedAt.Time,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	idStr := r.PathValue("id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		h.logger.Error("Invalid form ID", zap.String("id", idStr), zap.Error(err))
		http.Error(w, "Invalid form ID", http.StatusBadRequest)
		return
	}

	err = h.store.Delete(ctx, id)
	if err != nil {
		h.logger.Error("Failed to delete form", zap.Error(err))
		http.Error(w, "Failed to delete form", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
