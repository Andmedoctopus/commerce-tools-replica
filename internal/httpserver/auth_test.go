package httpserver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"commercetools-replica/internal/domain"
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

func TestSignupHandler_Created(t *testing.T) {
	gin.SetMode(gin.TestMode)
	proj := &domain.Project{ID: "proj-id", Key: "proj-key"}
	projectRepo := &stubProjectRepo{project: proj}
	authSvc := &stubCustomerAuthSvc{
		customer: &domain.Customer{ID: "cust-id", ProjectID: proj.ID, Email: "user@example.com"},
	}
	router, err := buildRouter(logDiscard(), nil, Deps{
		ProjectRepo: projectRepo,
		ProductSvc:  &stubProductService{},
		CartSvc:     &stubCartService{},
		CategorySvc: &stubCategoryService{},
		CustomerSvc: authSvc,
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
		ProjectRepo: projectRepo,
		ProductSvc:  &stubProductService{},
		CartSvc:     &stubCartService{},
		CategorySvc: &stubCategoryService{},
		CustomerSvc: authSvc,
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
		ProjectRepo: projectRepo,
		ProductSvc:  &stubProductService{},
		CartSvc:     &stubCartService{},
		CategorySvc: &stubCategoryService{},
		CustomerSvc: authSvc,
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
		ProjectRepo: projectRepo,
		ProductSvc:  &stubProductService{},
		CartSvc:     &stubCartService{},
		CategorySvc: &stubCategoryService{},
		CustomerSvc: authSvc,
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
