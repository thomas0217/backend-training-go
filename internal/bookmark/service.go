package bookmark

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type Query interface {
	Create(ctx context.Context, arg CreateParams) (Bookmark, error)
	Delete(ctx context.Context, arg DeleteParams) error
	Exist(ctx context.Context, arg ExistParams) (bool, error)
	GetFormsByUserID(ctx context.Context, userID uuid.UUID) ([]GetFormsByUserIDRow, error)
	CountByFormID(ctx context.Context, formID uuid.UUID) (int64, error)
	CountByUserID(ctx context.Context, userID uuid.UUID) (int64, error)
}
type Service struct {
	logger  *zap.Logger
	queries Query
}

func NewService(logger *zap.Logger, db *pgxpool.Pool) *Service {
	return &Service{
		logger:  logger,
		queries: New(db),
	}
}
func NewServiceWithQuery(logger *zap.Logger, q Query) *Service {
	return &Service{
		logger:  logger,
		queries: q,
	}
}

func (s Service) CountByFormID(ctx context.Context, formID uuid.UUID) (int64, error) {
	count, err := s.queries.CountByFormID(ctx, formID)
	if err != nil {
		s.logger.Error("failed to count bookmarks", zap.Error(err))
		return 0, err
	}
	return count, nil
}

func (s Service) CountByUserID(ctx context.Context, userID uuid.UUID) (int64, error) {
	count, err := s.queries.CountByUserID(ctx, userID)
	if err != nil {
		s.logger.Error("failed to count bookmarks", zap.Error(err))
		return 0, err
	}
	return count, nil
}

func (s Service) GetFormsByUserID(ctx context.Context, userID uuid.UUID) ([]GetFormsByUserIDRow, error) {
	forms, err := s.queries.GetFormsByUserID(ctx, userID)
	if err != nil {
		s.logger.Error("Failed to get bookmarked forms", zap.Error(err))
		return nil, err
	}

	return forms, nil
}

func (s Service) ToggleBookmark(ctx context.Context, userID, formID uuid.UUID) (bool, error) {
	exists, err := s.queries.Exist(ctx, ExistParams{
		UserID: userID,
		FormID: formID,
	})
	if err != nil {
		s.logger.Error("Failed to check bookmark existence", zap.Error(err))
		return false, err
	}
	if exists {
		err = s.queries.Delete(ctx, DeleteParams{
			UserID: userID,
			FormID: formID,
		})
		if err != nil {
			s.logger.Error("Failed to remove bookmark", zap.Error(err))
			return false, err
		}
		s.logger.Info("Removed bookmark", zap.String("user_id", userID.String()), zap.String("form_id", formID.String()))
		return false, nil
	} else {
		_, err = s.queries.Create(ctx, CreateParams{
			UserID: userID,
			FormID: formID,
		})
		if err != nil {
			s.logger.Error("Failed to add bookmark", zap.Error(err))
			return false, err
		}
		s.logger.Info("Added bookmark", zap.String("user_id", userID.String()), zap.String("form_id", formID.String()))
		return true, nil
	}
}
