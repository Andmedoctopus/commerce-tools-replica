package cart

import (
	"context"
	"errors"
	"testing"

	"commercetools-replica/internal/domain"
	cartrepo "commercetools-replica/internal/repository/cart"
)

type stubRepo struct {
	createCart        *domain.Cart
	createErr         error
	getByIDResults    []*domain.Cart
	getByIDErr        error
	getByIDCalls      int
	activeCart        *domain.Cart
	activeErr         error
	addLineItemErr    error
	changeLineItemErr error
	lastAddCartID     string
	lastAddProduct    domain.Product
	lastAddQty        int
	lastAddSnapshot   map[string]interface{}
	lastChangeCartID  string
	lastChangeLineID  string
	lastChangeQty     int
	lastStateProject  string
	lastStateCartID   string
	lastStateValue    string
	setStateErr       error
}

func (s *stubRepo) Create(_ context.Context, _ cartrepo.CreateCartInput) (*domain.Cart, error) {
	return s.createCart, s.createErr
}

func (s *stubRepo) GetByID(_ context.Context, _, _ string) (*domain.Cart, error) {
	if s.getByIDErr != nil {
		return nil, s.getByIDErr
	}
	var res *domain.Cart
	if len(s.getByIDResults) > 0 {
		idx := s.getByIDCalls
		if idx >= len(s.getByIDResults) {
			idx = len(s.getByIDResults) - 1
		}
		res = s.getByIDResults[idx]
	}
	s.getByIDCalls++
	return res, nil
}

func (s *stubRepo) GetActiveByCustomer(_ context.Context, _, _ string) (*domain.Cart, error) {
	return s.activeCart, s.activeErr
}

func (s *stubRepo) GetActiveByAnonymous(_ context.Context, _, _ string) (*domain.Cart, error) {
	return s.activeCart, s.activeErr
}

func (s *stubRepo) AssignCustomerToAnonymous(_ context.Context, _, _, _ string) (*domain.Cart, error) {
	return nil, nil
}

func (s *stubRepo) AddLineItem(_ context.Context, cartID string, product domain.Product, quantity int, snapshot map[string]interface{}) error {
	s.lastAddCartID = cartID
	s.lastAddProduct = product
	s.lastAddQty = quantity
	s.lastAddSnapshot = snapshot
	return s.addLineItemErr
}

func (s *stubRepo) ChangeLineItemQuantity(_ context.Context, cartID, lineItemID string, quantity int) error {
	s.lastChangeCartID = cartID
	s.lastChangeLineID = lineItemID
	s.lastChangeQty = quantity
	return s.changeLineItemErr
}

func (s *stubRepo) SetState(_ context.Context, projectID, cartID, state string) error {
	s.lastStateProject = projectID
	s.lastStateCartID = cartID
	s.lastStateValue = state
	return s.setStateErr
}

type stubProductRepo struct {
	product     *domain.Product
	err         error
	lastProject string
	lastSKU     string
}

func (s *stubProductRepo) GetBySKU(_ context.Context, projectID, sku string) (*domain.Product, error) {
	s.lastProject = projectID
	s.lastSKU = sku
	return s.product, s.err
}

func strPtr(v string) *string {
	return &v
}

func TestServiceCreateValidation(t *testing.T) {
	svc := &Service{repo: &stubRepo{}}
	_, err := svc.Create(context.Background(), "proj", CreateInput{Currency: "   "})
	if err == nil || err.Error() != "currency required" {
		t.Fatalf("expected currency validation error, got %v", err)
	}
}

func TestServiceCreateHappyPath(t *testing.T) {
	expected := &domain.Cart{ID: "c1", ProjectID: "proj", Currency: "USD"}
	svc := &Service{repo: &stubRepo{createCart: expected}}
	got, err := svc.Create(context.Background(), "proj", CreateInput{Currency: "USD"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != expected {
		t.Fatalf("unexpected cart: %+v", got)
	}
}

func TestServiceCreateRepoError(t *testing.T) {
	svc := &Service{repo: &stubRepo{createErr: errors.New("boom")}}
	_, err := svc.Create(context.Background(), "proj", CreateInput{Currency: "USD"})
	if err == nil || err.Error() != "boom" {
		t.Fatalf("expected repo error, got %v", err)
	}
}

func TestServiceGetActive(t *testing.T) {
	expected := &domain.Cart{ID: "c1"}
	svc := &Service{repo: &stubRepo{activeCart: expected}}
	got, err := svc.GetActive(context.Background(), "proj", "cust")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != expected {
		t.Fatalf("unexpected cart: %+v", got)
	}
}

func TestServiceUpdateRequiresActions(t *testing.T) {
	svc := &Service{repo: &stubRepo{}}
	_, err := svc.Update(context.Background(), "proj", "cust", "cart", UpdateInput{})
	if err == nil || err.Error() != "actions required" {
		t.Fatalf("expected actions error, got %v", err)
	}
}

func TestServiceUpdateCustomerMismatch(t *testing.T) {
	repo := &stubRepo{getByIDResults: []*domain.Cart{{ID: "cart", CustomerID: strPtr("other")}}}
	svc := &Service{repo: repo}
	_, err := svc.Update(context.Background(), "proj", "cust", "cart", UpdateInput{
		Actions: []UpdateAction{{Action: "addLineItem", SKU: "sku", Quantity: 1}},
	})
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
}

func TestServiceUpdateAddLineItemValidation(t *testing.T) {
	repo := &stubRepo{getByIDResults: []*domain.Cart{{ID: "cart", CustomerID: strPtr("cust")}}}
	svc := &Service{repo: repo, productRepo: &stubProductRepo{}}

	_, err := svc.Update(context.Background(), "proj", "cust", "cart", UpdateInput{
		Actions: []UpdateAction{{Action: "addLineItem", SKU: "", Quantity: 1}},
	})
	if err == nil || err.Error() != "sku required" {
		t.Fatalf("expected sku error, got %v", err)
	}

	_, err = svc.Update(context.Background(), "proj", "cust", "cart", UpdateInput{
		Actions: []UpdateAction{{Action: "addLineItem", SKU: "sku", Quantity: 0}},
	})
	if err == nil || err.Error() != "quantity must be positive" {
		t.Fatalf("expected quantity error, got %v", err)
	}
}

func TestServiceUpdateAddLineItemProductErrors(t *testing.T) {
	repo := &stubRepo{getByIDResults: []*domain.Cart{{ID: "cart", CustomerID: strPtr("cust")}}}
	svc := &Service{repo: repo}
	_, err := svc.Update(context.Background(), "proj", "cust", "cart", UpdateInput{
		Actions: []UpdateAction{{Action: "addLineItem", SKU: "sku", Quantity: 1}},
	})
	if err == nil || err.Error() != "product repository unavailable" {
		t.Fatalf("expected product repo error, got %v", err)
	}

	productRepo := &stubProductRepo{err: domain.ErrNotFound}
	svc = &Service{repo: repo, productRepo: productRepo}
	_, err = svc.Update(context.Background(), "proj", "cust", "cart", UpdateInput{
		Actions: []UpdateAction{{Action: "addLineItem", SKU: "sku", Quantity: 1}},
	})
	if err == nil || err.Error() != "product not found" {
		t.Fatalf("expected product not found, got %v", err)
	}
}

func TestServiceUpdateAddLineItemRepoError(t *testing.T) {
	repo := &stubRepo{
		getByIDResults: []*domain.Cart{{ID: "cart", CustomerID: strPtr("cust")}},
		addLineItemErr: errors.New("add failed"),
	}
	product := &domain.Product{ID: "p1", SKU: "sku", Name: "Prod", PriceCents: 100, Currency: "USD"}
	svc := &Service{repo: repo, productRepo: &stubProductRepo{product: product}}
	_, err := svc.Update(context.Background(), "proj", "cust", "cart", UpdateInput{
		Actions: []UpdateAction{{Action: "addLineItem", SKU: "sku", Quantity: 2}},
	})
	if err == nil || err.Error() != "add failed" {
		t.Fatalf("expected repo error, got %v", err)
	}
}

func TestServiceUpdateAddLineItemSuccess(t *testing.T) {
	initial := &domain.Cart{ID: "cart", CustomerID: strPtr("cust")}
	updated := &domain.Cart{ID: "cart", CustomerID: strPtr("cust")}
	repo := &stubRepo{getByIDResults: []*domain.Cart{initial, updated}}
	product := &domain.Product{ID: "p1", SKU: "sku", Name: "Prod", PriceCents: 100, Currency: "USD"}
	svc := &Service{repo: repo, productRepo: &stubProductRepo{product: product}}
	got, err := svc.Update(context.Background(), "proj", "cust", "cart", UpdateInput{
		Actions: []UpdateAction{{Action: "addLineItem", SKU: "sku", Quantity: 2}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != updated {
		t.Fatalf("unexpected cart: %+v", got)
	}
	if repo.lastAddCartID != "cart" || repo.lastAddQty != 2 || repo.lastAddProduct.ID != "p1" {
		t.Fatalf("add line item not called as expected")
	}
}

func TestServiceUpdateChangeLineItemValidation(t *testing.T) {
	repo := &stubRepo{getByIDResults: []*domain.Cart{{ID: "cart", CustomerID: strPtr("cust")}}}
	svc := &Service{repo: repo}
	_, err := svc.Update(context.Background(), "proj", "cust", "cart", UpdateInput{
		Actions: []UpdateAction{{Action: "changeLineItemQuantity", LineItemID: "", Quantity: 1}},
	})
	if err == nil || err.Error() != "lineItemId required" {
		t.Fatalf("expected lineItemId error, got %v", err)
	}

	_, err = svc.Update(context.Background(), "proj", "cust", "cart", UpdateInput{
		Actions: []UpdateAction{{Action: "changeLineItemQuantity", LineItemID: "line", Quantity: 0}},
	})
	if err == nil || err.Error() != "quantity must be positive" {
		t.Fatalf("expected quantity error, got %v", err)
	}
}

func TestServiceUpdateChangeLineItemRepoError(t *testing.T) {
	repo := &stubRepo{
		getByIDResults:    []*domain.Cart{{ID: "cart", CustomerID: strPtr("cust")}},
		changeLineItemErr: errors.New("change failed"),
	}
	svc := &Service{repo: repo}
	_, err := svc.Update(context.Background(), "proj", "cust", "cart", UpdateInput{
		Actions: []UpdateAction{{Action: "changeLineItemQuantity", LineItemID: "line", Quantity: 2}},
	})
	if err == nil || err.Error() != "change failed" {
		t.Fatalf("expected repo error, got %v", err)
	}
}

func TestServiceUpdateChangeLineItemSuccess(t *testing.T) {
	initial := &domain.Cart{ID: "cart", CustomerID: strPtr("cust")}
	updated := &domain.Cart{ID: "cart", CustomerID: strPtr("cust")}
	repo := &stubRepo{getByIDResults: []*domain.Cart{initial, updated}}
	svc := &Service{repo: repo}
	got, err := svc.Update(context.Background(), "proj", "cust", "cart", UpdateInput{
		Actions: []UpdateAction{{Action: "changeLineItemQuantity", LineItemID: "line", Quantity: 3}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != updated {
		t.Fatalf("unexpected cart: %+v", got)
	}
	if repo.lastChangeCartID != "cart" || repo.lastChangeLineID != "line" || repo.lastChangeQty != 3 {
		t.Fatalf("change line item not called as expected")
	}
}

func TestServiceDeleteCustomerOwnership(t *testing.T) {
	repo := &stubRepo{getByIDResults: []*domain.Cart{{ID: "cart", CustomerID: strPtr("other")}}}
	svc := &Service{repo: repo}
	_, err := svc.Delete(context.Background(), "proj", "cust", "cart")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
}

func TestServiceDeleteCustomerHappyPath(t *testing.T) {
	initial := &domain.Cart{ID: "cart", CustomerID: strPtr("cust")}
	deleted := &domain.Cart{ID: "cart", CustomerID: strPtr("cust"), State: "deleted"}
	repo := &stubRepo{getByIDResults: []*domain.Cart{initial, deleted}}
	svc := &Service{repo: repo}
	got, err := svc.Delete(context.Background(), "proj", "cust", "cart")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != deleted {
		t.Fatalf("unexpected cart: %+v", got)
	}
	if repo.lastStateProject != "proj" || repo.lastStateCartID != "cart" || repo.lastStateValue != "deleted" {
		t.Fatalf("unexpected SetState args: %s %s %s", repo.lastStateProject, repo.lastStateCartID, repo.lastStateValue)
	}
}
