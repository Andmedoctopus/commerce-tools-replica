package cart

import (
	"context"
	"errors"
	"strings"

	"commercetools-replica/internal/domain"
	cartrepo "commercetools-replica/internal/repository/cart"
)

type Service struct {
	repo cartRepo
}

type cartRepo interface {
	Create(ctx context.Context, in cartrepo.CreateCartInput) (*domain.Cart, error)
	GetByID(ctx context.Context, projectID, id string) (*domain.Cart, error)
}

func New(repo cartrepo.Repository) *Service {
	return &Service{repo: repo}
}

type CreateInput struct {
	CustomerID *string `json:"customerId,omitempty"`
	Currency   string  `json:"currency"`
}

func (s *Service) Create(ctx context.Context, projectID string, in CreateInput) (*domain.Cart, error) {
	if strings.TrimSpace(in.Currency) == "" {
		return nil, errors.New("currency required")
	}
	return s.repo.Create(ctx, cartrepo.CreateCartInput{
		ProjectID:  projectID,
		CustomerID: in.CustomerID,
		Currency:   in.Currency,
	})
}

func (s *Service) Get(ctx context.Context, projectID, id string) (*domain.Cart, error) {
	return s.repo.GetByID(ctx, projectID, id)
}
