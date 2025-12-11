package product

import (
	"context"

	"commercetools-replica/internal/domain"
)

type Repository interface {
	ListByProject(ctx context.Context, projectID string) ([]domain.Product, error)
	GetByID(ctx context.Context, projectID, id string) (*domain.Product, error)
}
