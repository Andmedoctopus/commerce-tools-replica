package category

import (
	"context"

	"commercetools-replica/internal/domain"
	"commercetools-replica/internal/repository/category"
)

type Service struct {
	repo category.Repository
}

func New(repo category.Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) List(ctx context.Context, projectID string) ([]domain.Category, error) {
	return s.repo.ListByProject(ctx, projectID)
}

func (s *Service) Upsert(ctx context.Context, c domain.Category) (*domain.Category, error) {
	return s.repo.Upsert(ctx, c)
}
