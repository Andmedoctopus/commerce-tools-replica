package httpserver

import (
	"context"
	"errors"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"commercetools-replica/internal/domain"
	cartsvc "commercetools-replica/internal/service/cart"
	"github.com/gin-gonic/gin"
)

type stubProjectRepo struct {
	project *domain.Project
	err     error
}

func (s *stubProjectRepo) GetByKey(_ context.Context, _ string) (*domain.Project, error) {
	return s.project, s.err
}

func (s *stubProjectRepo) Create(_ context.Context, p *domain.Project) (*domain.Project, error) {
	return p, nil
}

func TestProjectMiddleware_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &stubProjectRepo{
		project: &domain.Project{ID: "123", Key: "proj", Name: "Test"},
	}
	router := gin.New()
	router.Use(projectMiddleware(logDiscard(), repo))
	router.GET("/projects/:projectKey/test", func(c *gin.Context) {
		pCtx := c.Request.Context().Value(projectCtxKey)
		if pCtx == nil {
			t.Fatalf("expected project in request context")
		}
		if _, ok := pCtx.(*domain.Project); !ok {
			t.Fatalf("unexpected project type in context")
		}
		pGin, exists := c.Get(string(projectCtxKey))
		if !exists || pGin == nil {
			t.Fatalf("expected project in gin context")
		}
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/projects/proj/test", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
}

func TestProjectMiddleware_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &stubProjectRepo{err: domain.ErrNotFound}
	router := gin.New()
	router.Use(projectMiddleware(logDiscard(), repo))
	router.GET("/projects/:projectKey/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/projects/missing/test", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", rec.Code)
	}
}

func TestProjectMiddleware_Error(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &stubProjectRepo{err: errors.New("boom")}
	router := gin.New()
	router.Use(projectMiddleware(logDiscard(), repo))
	router.GET("/projects/:projectKey/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/projects/proj/test", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", rec.Code)
	}
}

func TestProjectMiddleware_MissingKey(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &stubProjectRepo{}
	router := gin.New()
	router.Use(projectMiddleware(logDiscard(), repo))
	router.GET("/projects/:projectKey/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/projects//test", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}
}

type stubProductService struct {
	listResult []domain.Product
	getResult  *domain.Product
	err        error
}

func (s *stubProductService) List(_ context.Context, _ string) ([]domain.Product, error) {
	return s.listResult, s.err
}

func (s *stubProductService) Get(_ context.Context, _ string, _ string) (*domain.Product, error) {
	return s.getResult, s.err
}

type stubCartService struct{}

func (s *stubCartService) Create(_ context.Context, _ string, _ cartsvc.CreateInput) (*domain.Cart, error) {
	return nil, nil
}

func (s *stubCartService) Get(_ context.Context, _ string, _ string) (*domain.Cart, error) {
	return nil, nil
}

func TestProductsHandler_List(t *testing.T) {
	gin.SetMode(gin.TestMode)
	proj := &domain.Project{ID: "proj-id", Key: "proj-key"}
	projectRepo := &stubProjectRepo{project: proj}
	productSvc := &stubProductService{
		listResult: []domain.Product{
			{ID: "p1", ProjectID: proj.ID, Name: "Demo", Key: "demo", SKU: "SKU1", PriceCents: 100, Currency: "EUR"},
		},
	}
	cartSvc := &stubCartService{}
	router, err := buildRouter(logDiscard(), nil, Deps{
		ProjectRepo: projectRepo,
		ProductSvc:  productSvc,
		CartSvc:     cartSvc,
	})
	if err != nil {
		t.Fatalf("build router: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/proj-key/products", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if body := rec.Body.String(); !strings.Contains(body, `"masterData"`) {
		t.Fatalf("expected commercetools shape, got %q", body)
	}
}

func TestProductsHandler_Get_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	proj := &domain.Project{ID: "proj-id", Key: "proj-key"}
	projectRepo := &stubProjectRepo{project: proj}
	productSvc := &stubProductService{
		err: domain.ErrNotFound,
	}
	cartSvc := &stubCartService{}
	router, err := buildRouter(logDiscard(), nil, Deps{
		ProjectRepo: projectRepo,
		ProductSvc:  productSvc,
		CartSvc:     cartSvc,
	})
	if err != nil {
		t.Fatalf("build router: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/proj-key/products/abc", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", rec.Code)
	}
}

// CT-style prefix is the default path shape; covered by the list test above.

func TestProductsHandler_Search(t *testing.T) {
	gin.SetMode(gin.TestMode)
	proj := &domain.Project{ID: "proj-id", Key: "proj-key"}
	projectRepo := &stubProjectRepo{project: proj}
	productSvc := &stubProductService{
		listResult: []domain.Product{
			{ID: "b-id", ProjectID: proj.ID, Name: "Beta", Key: "b", SKU: "SKU2", PriceCents: 100, Currency: "EUR"},
			{ID: "a-id", ProjectID: proj.ID, Name: "Alpha", Key: "a", SKU: "SKU1", PriceCents: 200, Currency: "EUR"},
		},
	}
	cartSvc := &stubCartService{}
	router, err := buildRouter(logDiscard(), nil, Deps{
		ProjectRepo: projectRepo,
		ProductSvc:  productSvc,
		CartSvc:     cartSvc,
	})
	if err != nil {
		t.Fatalf("build router: %v", err)
	}

	body := `{"limit":1,"offset":0,"query":{"filter":[{"range":{"field":"variants.prices.centAmount","fieldType":"long","gte":0,"lte":150}}]}}`
	req := httptest.NewRequest(http.MethodPost, "/proj-key/products/search", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"total":1`) || !strings.Contains(rec.Body.String(), `"id":"b-id"`) {
		t.Fatalf("unexpected search response: %s", rec.Body.String())
	}
}

func TestProductsHandler_SearchSortByPriceDesc(t *testing.T) {
	gin.SetMode(gin.TestMode)
	proj := &domain.Project{ID: "proj-id", Key: "proj-key"}
	projectRepo := &stubProjectRepo{project: proj}
	productSvc := &stubProductService{
		listResult: []domain.Product{
			{ID: "cheap", ProjectID: proj.ID, Name: "Cheap", Key: "c", SKU: "SKU1", PriceCents: 100, Currency: "EUR"},
			{ID: "exp", ProjectID: proj.ID, Name: "Expensive", Key: "e", SKU: "SKU2", PriceCents: 500, Currency: "EUR"},
		},
	}
	cartSvc := &stubCartService{}
	router, err := buildRouter(logDiscard(), nil, Deps{
		ProjectRepo: projectRepo,
		ProductSvc:  productSvc,
		CartSvc:     cartSvc,
	})
	if err != nil {
		t.Fatalf("build router: %v", err)
	}

	body := `{"limit":2,"offset":0,"sort":[{"field":"variants.prices.centAmount","order":"desc"}]}`
	req := httptest.NewRequest(http.MethodPost, "/proj-key/products/search", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	bodyStr := rec.Body.String()
	firstIdx := strings.Index(bodyStr, `"id":"`)
	secondIdx := strings.Index(bodyStr[firstIdx+1:], `"id":"`)
	if firstIdx == -1 || secondIdx == -1 {
		t.Fatalf("unexpected search response: %s", bodyStr)
	}
	if !strings.Contains(bodyStr, `"id":"exp"`) || firstIdx > strings.Index(bodyStr, `"id":"exp"`) {
		t.Fatalf("expected expensive first in desc price sort, got %s", bodyStr)
	}
}

func logDiscard() *log.Logger {
	return log.New(io.Discard, "", 0)
}
