package cart

import (
	"context"
	"errors"
	"testing"

	"commercetools-replica/internal/domain"
	cartrepo "commercetools-replica/internal/repository/cart"
)

type stubRepo struct {
	createCart *domain.Cart
	createErr  error
}

func (s *stubRepo) Create(_ context.Context, _ cartrepo.CreateCartInput) (*domain.Cart, error) {
	return s.createCart, s.createErr
}

func (s *stubRepo) GetByID(_ context.Context, _, _ string) (*domain.Cart, error) {
	return nil, nil
}

func (s *stubRepo) GetActiveByCustomer(_ context.Context, _, _ string) (*domain.Cart, error) {
	return nil, nil
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
