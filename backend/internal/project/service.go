package project

import (
	"context"

	"github.com/google/uuid"

	"taskflow/internal/models"
)

type UpdateProjectInput struct {
	Name             *string
	Description      *string
	ClearDescription bool
}

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) List(ctx context.Context, userID uuid.UUID) ([]models.Project, error) {
	return s.repo.List(ctx, userID)
}

func (s *Service) Create(ctx context.Context, userID uuid.UUID, name string, description *string) (*models.Project, error) {
	if name == "" {
		return nil, &models.ValidationError{
			Fields: map[string]string{"name": "is required"},
		}
	}

	p := &models.Project{
		Name:        name,
		Description: description,
		OwnerID:     userID,
	}

	if err := s.repo.Create(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*models.Project, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) Update(ctx context.Context, id, userID uuid.UUID, input UpdateProjectInput) (*models.Project, error) {
	project, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if project.OwnerID != userID {
		return nil, models.ErrForbidden
	}

	return s.repo.Update(ctx, id, input)
}

func (s *Service) Delete(ctx context.Context, id, userID uuid.UUID) error {
	project, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if project.OwnerID != userID {
		return models.ErrForbidden
	}

	return s.repo.Delete(ctx, id)
}
