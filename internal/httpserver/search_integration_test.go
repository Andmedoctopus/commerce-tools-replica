package httpserver

import (
	"context"
	"log"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"commercetools-replica/internal/domain"
	"commercetools-replica/internal/migrate"
	categoryrepo "commercetools-replica/internal/repository/category"
	productrepo "commercetools-replica/internal/repository/product"
	categorysvc "commercetools-replica/internal/service/category"
	productsvc "commercetools-replica/internal/service/product"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestSearchHandler_IntegrationFiltersByCategory(t *testing.T) {
	ctx := context.Background()
	pool := searchIntegrationPool(ctx, t)
	defer pool.Close()
	if err := migrate.Apply(ctx, pool); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	resetSearchTables(ctx, t, pool)

	var projectID string
	if err := pool.QueryRow(ctx, `INSERT INTO projects (key, name) VALUES ('proj-key', 'Proj') RETURNING id::text`).Scan(&projectID); err != nil {
		t.Fatalf("insert project: %v", err)
	}

	catRepo := categoryrepo.NewPostgres(pool)
	catSvc := categorysvc.New(catRepo)
	cat, err := catSvc.Upsert(ctx, domain.Category{
		ProjectID: projectID,
		Key:       "cactus",
		Name:      "Cactus",
	})
	if err != nil {
		t.Fatalf("upsert category: %v", err)
	}

	prodRepo := productrepo.NewPostgres(pool, log.New(os.Stdout, "[test] ", log.LstdFlags))
	prodSvc := productsvc.New(prodRepo)

	_, err = prodRepo.Upsert(ctx, domain.Product{
		ProjectID:  projectID,
		Key:        "p1",
		SKU:        "SKU1",
		Name:       "With Cat",
		PriceCents: 100,
		Currency:   "EUR",
		Attributes: map[string]interface{}{"categories": []string{cat.Key}},
	})
	if err != nil {
		t.Fatalf("upsert product with cat: %v", err)
	}
	_, err = prodRepo.Upsert(ctx, domain.Product{
		ProjectID:  projectID,
		Key:        "p2",
		SKU:        "SKU2",
		Name:       "No Cat",
		PriceCents: 50,
		Currency:   "EUR",
		Attributes: map[string]interface{}{"categories": []string{"other"}},
	})
	if err != nil {
		t.Fatalf("upsert product without cat: %v", err)
	}

	gin.SetMode(gin.TestMode)
	router, err := buildRouter(logDiscard(), pool, Deps{
		ProjectRepo: &stubProjectRepo{project: &domain.Project{ID: projectID, Key: "proj-key"}},
		ProductSvc:  prodSvc,
		CartSvc:     &stubCartService{},
		CategorySvc: catSvc,
		CustomerSvc: &stubCustomerService{customer: &domain.Customer{ID: "cust", ProjectID: projectID}},
	})
	if err != nil {
		t.Fatalf("build router: %v", err)
	}

	body := `{"query":{"filter":[{"exact":{"field":"categories","value":"` + cat.ID + `"}}]}}`
	req := httptest.NewRequest("POST", "/proj-key/products/search", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"id":"`) || !strings.Contains(rec.Body.String(), `"total":1`) {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "p2") {
		t.Fatalf("expected to exclude product without category, got %s", rec.Body.String())
	}
}

func searchIntegrationPool(ctx context.Context, t *testing.T) *pgxpool.Pool {
	t.Helper()
	candidates := []string{
		os.Getenv("TEST_DB_DSN"),
		"postgres://commerce:commerce@db-test:5432/commerce_test?sslmode=disable",
		"postgres://commerce:commerce@localhost:5433/commerce_test?sslmode=disable",
	}
	var lastErr error
	for _, dsn := range candidates {
		if dsn == "" {
			continue
		}
		pool, err := pgxpool.New(ctx, dsn)
		if err != nil {
			lastErr = err
			continue
		}
		if err := pool.Ping(ctx); err != nil {
			lastErr = err
			pool.Close()
			continue
		}
		return pool
	}
	t.Fatalf("connect db: %v", lastErr)
	return nil
}

func resetSearchTables(ctx context.Context, t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	if _, err := pool.Exec(ctx, `TRUNCATE cart_lines, carts, products, customers, projects, categories RESTART IDENTITY CASCADE`); err != nil {
		t.Fatalf("truncate tables: %v", err)
	}
}
