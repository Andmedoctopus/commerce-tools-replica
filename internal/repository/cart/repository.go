package cart

import (
	"context"

	"commercetools-replica/internal/domain"
)

type CreateCartInput struct {
	ProjectID  string
	CustomerID *string
	Currency   string
}

type Repository interface {
	Create(ctx context.Context, in CreateCartInput) (*domain.Cart, error)
	GetByID(ctx context.Context, projectID, id string) (*domain.Cart, error)
	GetActiveByCustomer(ctx context.Context, projectID, customerID string) (*domain.Cart, error)
}
