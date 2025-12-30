package cart

import (
	"context"

	"commercetools-replica/internal/domain"
)

type CreateCartInput struct {
	ProjectID   string
	CustomerID  *string
	AnonymousID *string
	Currency    string
}

type Repository interface {
	Create(ctx context.Context, in CreateCartInput) (*domain.Cart, error)
	GetByID(ctx context.Context, projectID, id string) (*domain.Cart, error)
	GetActiveByCustomer(ctx context.Context, projectID, customerID string) (*domain.Cart, error)
	GetActiveByAnonymous(ctx context.Context, projectID, anonymousID string) (*domain.Cart, error)
	AssignCustomerToAnonymous(ctx context.Context, projectID, anonymousID, customerID string) (*domain.Cart, error)
	AddLineItem(ctx context.Context, cartID string, product domain.Product, quantity int, snapshot map[string]interface{}) error
	ChangeLineItemQuantity(ctx context.Context, cartID, lineItemID string, quantity int) error
	SetState(ctx context.Context, projectID, cartID, state string) error
}
