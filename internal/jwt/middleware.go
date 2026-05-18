package jwt

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type contextKey string

const UserContextKey contextKey = "user-id"

type Verifier interface {
	Parse(ctx context.Context, tokenString string) (uuid.UUID, error)
}

type Middleware struct {
	logger   *zap.Logger
	verifier Verifier
}

func NewMiddleware(logger *zap.Logger, verifier Verifier) Middleware {
	return Middleware{
		logger:   logger,
		verifier: verifier,
	}
}

func (m Middleware) HandlerFunc(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		token := r.Header.Get("Authorization")
		if token == "" {
			m.logger.Warn("Authorization header required")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		userID, err := m.verifier.Parse(ctx, token)
		if err != nil {
			m.logger.Warn("Authorization header invalid", zap.Error(err))
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		ctx = context.WithValue(ctx, UserContextKey, userID)

		m.logger.Debug("Authorization header valid", zap.String("user_id", userID.String()))
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}
