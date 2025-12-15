package category

import (
	"context"

	"commercetools-replica/internal/domain"
)

type Repository interface {
	ListByProject(ctx context.Context, projectID string) ([]domain.Category, error)
	Upsert(ctx context.Context, c domain.Category) (*domain.Category, error)
}
