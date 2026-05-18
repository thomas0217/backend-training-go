package auth

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

const UserContextKey = "user-id"

type UserValidator interface {
	Exists(ctx context.Context, id uuid.UUID) (bool, error)
}
type Middleware struct {
	logger    *zap.Logger
	validator UserValidator
}

func NewMiddleware(logger *zap.Logger, validator UserValidator) *Middleware {
	return &Middleware{
		logger:    logger,
		validator: validator,
	}
}

func (m Middleware) AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		// 從request拿value
		token := r.Header.Get("Authorization")
		// 取是uuid的值
		id, err := uuid.Parse(token)
		// not UUID
		if err != nil {
			m.logger.Warn("Invalid Authorization token", zap.String("token", token))
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		ctx := r.Context()
		_, err = m.validator.Exists(ctx, id)
		if err != nil {
			m.logger.Warn("User does not exist", zap.String("user_id", id.String()))
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		ctx = context.WithValue(ctx, UserContextKey, token)
		next(w, r.WithContext(ctx))
	}
}
