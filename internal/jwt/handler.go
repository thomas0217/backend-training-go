package jwt

import (
	"context"
	"encoding/json"
	"net/http"

	"go.uber.org/zap"
)

type Request struct {
	RefreshToken string `json:"refresh_token"`
}

type Response struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type Refresher interface {
	Refresh(ctx context.Context, refreshToken string) (string, string, error)
}

type Handler struct {
	logger    *zap.Logger
	refresher Refresher
}

func NewHandler(Logger *zap.Logger, Refresher Refresher) *Handler {
	return &Handler{
		logger:    Logger,
		refresher: Refresher,
	}
}

func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req Request

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Warn("Failed to decode refresh request body", zap.Error(err))
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	// handler裡面有refresher這個欄位 refresher有Refresh這個由service.go實作的func
	accessToken, newRefreshToken, err := h.refresher.Refresh(r.Context(), req.RefreshToken)
	if err != nil {
		h.logger.Warn("Failed to refresh token", zap.Error(err))
		// 內場說有問題 (過期/找不到)。注意：為了安全，對外只統一回 401，不說具體原因
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	res := Response{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
	}

	// http header設為傳json
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(res); err != nil {
		h.logger.Error("Failed to encode response", zap.Error(err))
	}
}

var (
	_ = context.Background
	_ = json.NewDecoder
	_ = http.StatusOK
	_ = zap.NewNop
)
