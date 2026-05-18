package user

import (
	"context"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type Querier interface {
	Create(ctx context.Context, email string) (User, error)
	Exists(ctx context.Context, id uuid.UUID) (uuid.UUID, error)
	ExistsUser(ctx context.Context, email string) (bool, error)
	GetByEmail(ctx context.Context, email string) (User, error)
}

type Service struct {
	logger  *zap.Logger
	querier Querier
}

func NewService(logger *zap.Logger, db DBTX) *Service {
	return &Service{
		logger:  logger,
		querier: New(db),
	}
}

func (s *Service) Create(ctx context.Context, email string) (User, error) {
	result, err := s.querier.Create(ctx, email)
	if err != nil {
		s.logger.Error("Failed to create user", zap.Error(err))
		return User{}, err
	}
	s.logger.Info("created user", zap.String("email", email))
	return result, nil
}

func (s *Service) Exists(ctx context.Context, id uuid.UUID) (bool, error) {
	_, err := s.querier.Exists(ctx, id)
	if err != nil {
		s.logger.Error("Failed to check if user exists", zap.Error(err))
		return false, err
	}
	return true, nil
}

func (s *Service) ExistsUser(ctx context.Context, email string) (bool, error) {
	result, err := s.querier.ExistsUser(ctx, email)
	if err != nil {
		s.logger.Error("Failed to check if user exists", zap.Error(err))
		return false, err
	}
	return result, nil
}

// GetByEmail 用 email 查出整筆 user（含 id）
// 用於 OAuth callback：登入成功後拿到 email，要找出對應的 user.ID 才能簽 token
func (s *Service) GetByEmail(ctx context.Context, email string) (User, error) {
	result, err := s.querier.GetByEmail(ctx, email)
	if err != nil {
		s.logger.Error("Failed to get user by email", zap.Error(err))
		return User{}, err
	}
	return result, nil
}
