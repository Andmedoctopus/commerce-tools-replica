package customer

import (
	"context"

	"commercetools-replica/internal/domain"
)

// Repository persists and fetches customers.
type Repository interface {
	Create(ctx context.Context, c domain.Customer) (*domain.Customer, error)
	GetByEmail(ctx context.Context, projectID, email string) (*domain.Customer, error)
	GetByID(ctx context.Context, projectID, id string) (*domain.Customer, error)
}
