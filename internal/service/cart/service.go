package cart

import (
	"context"
	"errors"
	"strings"

	"commercetools-replica/internal/domain"
	cartrepo "commercetools-replica/internal/repository/cart"
)

type Service struct {
	repo        cartRepo
	productRepo productRepo
}

type cartRepo interface {
	Create(ctx context.Context, in cartrepo.CreateCartInput) (*domain.Cart, error)
	GetByID(ctx context.Context, projectID, id string) (*domain.Cart, error)
	GetActiveByCustomer(ctx context.Context, projectID, customerID string) (*domain.Cart, error)
	GetActiveByAnonymous(ctx context.Context, projectID, anonymousID string) (*domain.Cart, error)
	AssignCustomerToAnonymous(ctx context.Context, projectID, anonymousID, customerID string) (*domain.Cart, error)
	AddLineItem(ctx context.Context, cartID string, product domain.Product, quantity int, snapshot map[string]interface{}) error
	ChangeLineItemQuantity(ctx context.Context, cartID, lineItemID string, quantity int) error
}

type productRepo interface {
	GetBySKU(ctx context.Context, projectID, sku string) (*domain.Product, error)
}

func New(repo cartrepo.Repository, productRepo productRepo) *Service {
	return &Service{repo: repo, productRepo: productRepo}
}

type CreateInput struct {
	CustomerID  *string `json:"customerId,omitempty"`
	AnonymousID *string `json:"anonymousId,omitempty"`
	Currency    string  `json:"currency"`
}

type UpdateInput struct {
	Version int            `json:"version"`
	Actions []UpdateAction `json:"actions"`
}

type UpdateAction struct {
	Action     string `json:"action"`
	SKU        string `json:"sku,omitempty"`
	LineItemID string `json:"lineItemId,omitempty"`
	Quantity   int    `json:"quantity,omitempty"`
}

func (s *Service) Create(ctx context.Context, projectID string, in CreateInput) (*domain.Cart, error) {
	if strings.TrimSpace(in.Currency) == "" {
		return nil, errors.New("currency required")
	}
	return s.repo.Create(ctx, cartrepo.CreateCartInput{
		ProjectID:   projectID,
		CustomerID:  in.CustomerID,
		AnonymousID: in.AnonymousID,
		Currency:    in.Currency,
	})
}

func (s *Service) Get(ctx context.Context, projectID, id string) (*domain.Cart, error) {
	return s.repo.GetByID(ctx, projectID, id)
}

func (s *Service) GetActive(ctx context.Context, projectID, customerID string) (*domain.Cart, error) {
	return s.repo.GetActiveByCustomer(ctx, projectID, customerID)
}

func (s *Service) GetActiveAnonymous(ctx context.Context, projectID, anonymousID string) (*domain.Cart, error) {
	return s.repo.GetActiveByAnonymous(ctx, projectID, anonymousID)
}

func (s *Service) AssignCustomerFromAnonymous(ctx context.Context, projectID, anonymousID, customerID string) (*domain.Cart, error) {
	return s.repo.AssignCustomerToAnonymous(ctx, projectID, anonymousID, customerID)
}

func (s *Service) Update(ctx context.Context, projectID, customerID, cartID string, in UpdateInput) (*domain.Cart, error) {
	return s.updateWithOwner(ctx, projectID, cartID, &customerID, nil, in)
}

func (s *Service) UpdateAnonymous(ctx context.Context, projectID, anonymousID, cartID string, in UpdateInput) (*domain.Cart, error) {
	return s.updateWithOwner(ctx, projectID, cartID, nil, &anonymousID, in)
}

func (s *Service) updateWithOwner(ctx context.Context, projectID, cartID string, customerID, anonymousID *string, in UpdateInput) (*domain.Cart, error) {
	if len(in.Actions) == 0 {
		return nil, errors.New("actions required")
	}
	cart, err := s.repo.GetByID(ctx, projectID, cartID)
	if err != nil {
		return nil, err
	}
	switch {
	case customerID != nil:
		if cart.CustomerID == nil || *cart.CustomerID != *customerID {
			return nil, domain.ErrNotFound
		}
	case anonymousID != nil:
		if cart.AnonymousID == nil || *cart.AnonymousID != *anonymousID {
			return nil, domain.ErrNotFound
		}
	default:
		return nil, domain.ErrNotFound
	}

	for _, action := range in.Actions {
		switch strings.ToLower(strings.TrimSpace(action.Action)) {
		case "addlineitem":
			sku := strings.TrimSpace(action.SKU)
			if sku == "" {
				return nil, errors.New("sku required")
			}
			if action.Quantity <= 0 {
				return nil, errors.New("quantity must be positive")
			}
			if s.productRepo == nil {
				return nil, errors.New("product repository unavailable")
			}
			product, err := s.productRepo.GetBySKU(ctx, projectID, sku)
			if err != nil {
				if errors.Is(err, domain.ErrNotFound) {
					return nil, errors.New("product not found")
				}
				return nil, err
			}
			snapshot := snapshotFromProduct(*product)
			if err := s.repo.AddLineItem(ctx, cartID, *product, action.Quantity, snapshot); err != nil {
				return nil, err
			}
		case "changelineitemquantity":
			lineID := strings.TrimSpace(action.LineItemID)
			if lineID == "" {
				return nil, errors.New("lineItemId required")
			}
			if action.Quantity <= 0 {
				return nil, errors.New("quantity must be positive")
			}
			if err := s.repo.ChangeLineItemQuantity(ctx, cartID, lineID, action.Quantity); err != nil {
				return nil, err
			}
		default:
			return nil, errors.New("unsupported action")
		}
	}

	return s.repo.GetByID(ctx, projectID, cartID)
}

func snapshotFromProduct(p domain.Product) map[string]interface{} {
	slug := strings.TrimSpace(p.Key)
	if slug == "" {
		slug = strings.ReplaceAll(strings.ToLower(p.Name), " ", "-")
	}
	snap := map[string]interface{}{
		"productKey":  p.Key,
		"productName": p.Name,
		"sku":         p.SKU,
		"productSlug": slug,
		"priceCents":  p.PriceCents,
		"currency":    p.Currency,
	}
	if len(p.Attributes) > 0 {
		if images, ok := p.Attributes["images"]; ok {
			snap["images"] = images
		}
	}
	return snap
}
