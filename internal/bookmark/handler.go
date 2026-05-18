package bookmark

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"awesomeProject/internal/jwt"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type Store interface {
	ToggleBookmark(ctx context.Context, userID, formID uuid.UUID) (bool, error)
	GetFormsByUserID(ctx context.Context, userID uuid.UUID) ([]GetFormsByUserIDRow, error)
	CountByUserID(ctx context.Context, userID uuid.UUID) (int64, error)
	CountByFormID(ctx context.Context, formID uuid.UUID) (int64, error)
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

// From sqlc GetFormsByUserIDRow
type BookmarkResponse struct {
	FormID      string    `json:"form_id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	AuthorID    string    `json:"author_id"`
	CreatedAt   time.Time `json:"created_at"`
}

// = GetFormsByUserIDRow
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, ok := ctx.Value(jwt.UserContextKey).(uuid.UUID)
	if !ok {
		h.logger.Error("Failed to get user ID from context")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	forms, err := h.store.GetFormsByUserID(ctx, userID)
	if err != nil {
		h.logger.Error("Failed to get bookmarked forms", zap.Error(err))
		http.Error(w, "Failed to get bookmarks", http.StatusInternalServerError)
		return
	}

	var resp []BookmarkResponse
	for _, f := range forms {
		resp = append(resp, BookmarkResponse{
			FormID:      f.FormID.String(),
			Title:       f.Title,
			Description: f.Description.String,
			AuthorID:    uuid.UUID(f.AuthorID.Bytes).String(),
			CreatedAt:   f.CreatedAt.Time,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		h.logger.Error("Failed to encode response", zap.Error(err))
	}
}

// switch between marked/ unmarked
func (h *Handler) Toggle(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, ok := ctx.Value(jwt.UserContextKey).(uuid.UUID)
	if !ok {
		h.logger.Error("Failed to get user ID from context")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	formIDStr := r.PathValue("form_id")
	formID, err := uuid.Parse(formIDStr)
	if err != nil {
		h.logger.Error("Invalid form ID", zap.String("form_id", formIDStr), zap.Error(err))
		http.Error(w, "Invalid form ID", http.StatusBadRequest)
		return
	}

	bookmarked, err := h.store.ToggleBookmark(ctx, userID, formID)
	if err != nil {
		h.logger.Error("Failed to toggle bookmark", zap.Error(err))
		http.Error(w, "Failed to toggle bookmark", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]bool{"bookmarked": bookmarked}); err != nil {
		h.logger.Error("Failed to encode response", zap.Error(err))
	}
}

func (h *Handler) CountByUserID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := ctx.Value(jwt.UserContextKey).(uuid.UUID)
	if !ok {
		h.logger.Error("Failed to get user ID from context")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	count, err := h.store.CountByUserID(ctx, userID)
	if err != nil {
		h.logger.Error("Failed to count bookmarks", zap.Error(err))
		http.Error(w, "Failed to count bookmarks", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(map[string]int64{"count": count}); err != nil {
		h.logger.Error("Failed to encode response", zap.Error(err))
	}
}

func (h *Handler) CountByFormID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	formIDStr := r.PathValue("form_id")
	formID, err := uuid.Parse(formIDStr)
	if err != nil {
		h.logger.Error("Invalid form ID", zap.String("form_id", formIDStr), zap.Error(err))
		http.Error(w, "Invalid form ID", http.StatusBadRequest)
		return
	}
	count, err := h.store.CountByFormID(ctx, formID)
	if err != nil {
		h.logger.Error("Failed to count bookmarks", zap.Error(err))
		http.Error(w, "Failed to count bookmarks", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]int64{"count": count}); err != nil {
		h.logger.Error("Failed to encode response", zap.Error(err))
	}
}
