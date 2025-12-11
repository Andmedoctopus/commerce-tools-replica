package product

import (
	"context"

	"commercetools-replica/internal/domain"
	productrepo "commercetools-replica/internal/repository/product"
)

type Service struct {
	repo productrepo.Repository
}

func New(repo productrepo.Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) List(ctx context.Context, projectID string) ([]domain.Product, error) {
	return s.repo.ListByProject(ctx, projectID)
}

func (s *Service) Get(ctx context.Context, projectID, id string) (*domain.Product, error) {
	return s.repo.GetByID(ctx, projectID, id)
}
