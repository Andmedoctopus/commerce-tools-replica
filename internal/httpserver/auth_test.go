package httpserver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"commercetools-replica/internal/domain"
	cartsvc "commercetools-replica/internal/service/cart"
	customersvc "commercetools-replica/internal/service/customer"
	"github.com/gin-gonic/gin"
)

type stubCustomerAuthSvc struct {
	customer *domain.Customer
	loginErr error
	signErr  error
	meErr    error
}

func (s *stubCustomerAuthSvc) Signup(_ context.Context, _ string, _ customersvc.SignupInput) (*domain.Customer, error) {
	return s.customer, s.signErr
}

func (s *stubCustomerAuthSvc) Login(_ context.Context, _ string, _ string, _ string) (*domain.Customer, string, string, error) {
	return s.customer, "access", "refresh", s.loginErr
}

func (s *stubCustomerAuthSvc) LookupByToken(_ context.Context, _ string, _ string) (*domain.Customer, error) {
	return s.customer, s.meErr
}

func (s *stubCustomerAuthSvc) AccessTTLSeconds() int {
	return 3600
}

type stubLoginCartService struct {
	cart *domain.Cart
	err  error
}

func (s *stubLoginCartService) Create(_ context.Context, _ string, _ cartsvc.CreateInput) (*domain.Cart, error) {
	return nil, nil
}

func (s *stubLoginCartService) Get(_ context.Context, _ string, _ string) (*domain.Cart, error) {
	return nil, nil
}

func (s *stubLoginCartService) GetActive(_ context.Context, _ string, _ string) (*domain.Cart, error) {
	return s.cart, s.err
}

func (s *stubLoginCartService) Update(_ context.Context, _ string, _ string, _ string, _ cartsvc.UpdateInput) (*domain.Cart, error) {
	return nil, nil
}

func (s *stubLoginCartService) GetActiveAnonymous(_ context.Context, _ string, _ string) (*domain.Cart, error) {
	return nil, nil
}

func (s *stubLoginCartService) UpdateAnonymous(_ context.Context, _ string, _ string, _ string, _ cartsvc.UpdateInput) (*domain.Cart, error) {
	return nil, nil
}

func (s *stubLoginCartService) AssignCustomerFromAnonymous(_ context.Context, _ string, _ string, _ string) (*domain.Cart, error) {
	return nil, nil
}

func TestSignupHandler_Created(t *testing.T) {
	gin.SetMode(gin.TestMode)
	proj := &domain.Project{ID: "proj-id", Key: "proj-key"}
	projectRepo := &stubProjectRepo{project: proj}
	authSvc := &stubCustomerAuthSvc{
		customer: &domain.Customer{ID: "cust-id", ProjectID: proj.ID, Email: "user@example.com"},
	}
	router, err := buildRouter(logDiscard(), nil, Deps{
		ProjectRepo:  projectRepo,
		ProductSvc:   &stubProductService{},
		CartSvc:      &stubCartService{},
		CategorySvc:  &stubCategoryService{},
		CustomerSvc:  authSvc,
		AnonymousSvc: &stubAnonymousService{},
	})
	if err != nil {
		t.Fatalf("build router: %v", err)
	}

	body := `{"email":"user@example.com","password":"Abcdefg1","addresses":[{"country":"US"}]}`
	req := httptest.NewRequest(http.MethodPost, "/proj-key/me/signup", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"email":"user@example.com"`) {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
}

func TestTokenHandler_InvalidCredentials(t *testing.T) {
	gin.SetMode(gin.TestMode)
	proj := &domain.Project{ID: "proj-id", Key: "proj-key"}
	projectRepo := &stubProjectRepo{project: proj}
	authSvc := &stubCustomerAuthSvc{
		loginErr: customersvc.ErrInvalidCredentials,
	}
	router, err := buildRouter(logDiscard(), nil, Deps{
		ProjectRepo:  projectRepo,
		ProductSvc:   &stubProductService{},
		CartSvc:      &stubCartService{},
		CategorySvc:  &stubCategoryService{},
		CustomerSvc:  authSvc,
		AnonymousSvc: &stubAnonymousService{},
	})
	if err != nil {
		t.Fatalf("build router: %v", err)
	}

	body := `grant_type=password&username=user%40example.com&password=badpass&scope=manage_project:proj-key`
	req := httptest.NewRequest(http.MethodPost, "/oauth/proj-key/customers/token", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestMeHandler_UnauthorizedWithoutToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	proj := &domain.Project{ID: "proj-id", Key: "proj-key"}
	projectRepo := &stubProjectRepo{project: proj}
	authSvc := &stubCustomerAuthSvc{}
	router, err := buildRouter(logDiscard(), nil, Deps{
		ProjectRepo:  projectRepo,
		ProductSvc:   &stubProductService{},
		CartSvc:      &stubCartService{},
		CategorySvc:  &stubCategoryService{},
		CustomerSvc:  authSvc,
		AnonymousSvc: &stubAnonymousService{},
	})
	if err != nil {
		t.Fatalf("build router: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/proj-key/me", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestMeHandler_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	proj := &domain.Project{ID: "proj-id", Key: "proj-key"}
	projectRepo := &stubProjectRepo{project: proj}
	authSvc := &stubCustomerAuthSvc{
		customer: &domain.Customer{ID: "cust-id", ProjectID: proj.ID, Email: "me@example.com"},
	}
	router, err := buildRouter(logDiscard(), nil, Deps{
		ProjectRepo:  projectRepo,
		ProductSvc:   &stubProductService{},
		CartSvc:      &stubCartService{},
		CategorySvc:  &stubCategoryService{},
		CustomerSvc:  authSvc,
		AnonymousSvc: &stubAnonymousService{},
	})
	if err != nil {
		t.Fatalf("build router: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/proj-key/me", nil)
	req.Header.Set("Authorization", "Bearer token")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"email":"me@example.com"`) {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
}

func TestLoginHandler_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	proj := &domain.Project{ID: "proj-id", Key: "proj-key"}
	projectRepo := &stubProjectRepo{project: proj}
	authSvc := &stubCustomerAuthSvc{
		customer: &domain.Customer{ID: "cust-id", ProjectID: proj.ID, Email: "user@example.com"},
	}
	cartSvc := &stubLoginCartService{err: domain.ErrNotFound}
	router, err := buildRouter(logDiscard(), nil, Deps{
		ProjectRepo:  projectRepo,
		ProductSvc:   &stubProductService{},
		CartSvc:      cartSvc,
		CategorySvc:  &stubCategoryService{},
		CustomerSvc:  authSvc,
		AnonymousSvc: &stubAnonymousService{},
	})
	if err != nil {
		t.Fatalf("build router: %v", err)
	}

	body := `{"email":"user@example.com","password":"secret"}`
	req := httptest.NewRequest(http.MethodPost, "/proj-key/me/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"customer"`) {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
}

func TestLoginHandler_InvalidBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	proj := &domain.Project{ID: "proj-id", Key: "proj-key"}
	projectRepo := &stubProjectRepo{project: proj}
	authSvc := &stubCustomerAuthSvc{}
	cartSvc := &stubLoginCartService{}
	router, err := buildRouter(logDiscard(), nil, Deps{
		ProjectRepo:  projectRepo,
		ProductSvc:   &stubProductService{},
		CartSvc:      cartSvc,
		CategorySvc:  &stubCategoryService{},
		CustomerSvc:  authSvc,
		AnonymousSvc: &stubAnonymousService{},
	})
	if err != nil {
		t.Fatalf("build router: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/proj-key/me/login", strings.NewReader("{"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestLoginHandler_InvalidCredentials(t *testing.T) {
	gin.SetMode(gin.TestMode)
	proj := &domain.Project{ID: "proj-id", Key: "proj-key"}
	projectRepo := &stubProjectRepo{project: proj}
	authSvc := &stubCustomerAuthSvc{
		loginErr: customersvc.ErrInvalidCredentials,
	}
	cartSvc := &stubLoginCartService{}
	router, err := buildRouter(logDiscard(), nil, Deps{
		ProjectRepo:  projectRepo,
		ProductSvc:   &stubProductService{},
		CartSvc:      cartSvc,
		CategorySvc:  &stubCategoryService{},
		CustomerSvc:  authSvc,
		AnonymousSvc: &stubAnonymousService{},
	})
	if err != nil {
		t.Fatalf("build router: %v", err)
	}

	body := `{"email":"user@example.com","password":"bad"}`
	req := httptest.NewRequest(http.MethodPost, "/proj-key/me/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", rec.Code, rec.Body.String())
	}
}
