package form

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
)

type Querier interface {
	Create(ctx context.Context, arg CreateParams) (Form, error)
	List(ctx context.Context, userID uuid.UUID) ([]ListRow, error)
	Update(ctx context.Context, arg UpdateParams) (Form, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type Service struct {
	logger  *zap.Logger
	queries Querier //掛載interface
}

func NewService(logger *zap.Logger, db DBTX) *Service {
	return &Service{
		logger:  logger,
		queries: New(db),
	}
}

func (s *Service) Create(ctx context.Context, name, description string, authorID uuid.UUID) (Form, error) {
	result, err := s.queries.Create(ctx, CreateParams{
		Title:       name,
		Description: pgtype.Text{String: description, Valid: true},
		AuthorID:    pgtype.UUID{Bytes: authorID, Valid: true},
	})
	if err != nil {
		s.logger.Error("Failed to create form", zap.Error(err))
		return Form{}, err
	}

	s.logger.Info("Created form", zap.String("form_id", result.ID.String()), zap.String("title", result.Title), zap.String("description", result.Description.String))

	return result, nil
}

func (s *Service) List(ctx context.Context, userID uuid.UUID) ([]ListRow, error) {
	forms, err := s.queries.List(ctx, userID)
	if err != nil {
		s.logger.Error("Failed to list forms", zap.Error(err))
		return nil, err
	}

	s.logger.Info("Listed forms", zap.Int("count", len(forms)))

	return forms, nil
}

func (s *Service) Update(ctx context.Context, id uuid.UUID, title, description string) (Form, error) {
	result, err := s.queries.Update(ctx, UpdateParams{
		ID:          id,
		Title:       title,
		Description: pgtype.Text{String: description, Valid: true},
	})
	if err != nil {
		s.logger.Error("Failed to update form", zap.Error(err), zap.String("form_id", id.String()))
		return Form{}, err
	}

	s.logger.Info("Updated form", zap.String("form_id", result.ID.String()), zap.String("title", result.Title))

	return result, nil
}

func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	err := s.queries.Delete(ctx, id)
	if err != nil {
		s.logger.Error("Failed to delete form", zap.Error(err), zap.String("form_id", id.String()))
		return err
	}

	s.logger.Info("Deleted form", zap.String("form_id", id.String()))

	return nil
}
