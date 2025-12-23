package httpserver

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"os"
	"testing"

	"commercetools-replica/internal/domain"
	"commercetools-replica/internal/migrate"
	categoryrepo "commercetools-replica/internal/repository/category"
	categorysvc "commercetools-replica/internal/service/category"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestCategoriesHandler_IntegrationReturnsHierarchy(t *testing.T) {
	ctx := context.Background()
	pool := categoriesPool(ctx, t)
	defer pool.Close()
	if err := migrate.Apply(ctx, pool); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	resetCategoryTables(ctx, t, pool)

	var projectID string
	if err := pool.QueryRow(ctx, `INSERT INTO projects (key, name) VALUES ('proj-key', 'Proj') RETURNING id::text`).Scan(&projectID); err != nil {
		t.Fatalf("insert project: %v", err)
	}

	catRepo := categoryrepo.NewPostgres(pool)
	catSvc := categorysvc.New(catRepo)
	root, err := catSvc.Upsert(ctx, domain.Category{
		ProjectID: projectID,
		Key:       "root",
		Name:      "Root",
	})
	if err != nil {
		t.Fatalf("upsert root: %v", err)
	}
	child, err := catSvc.Upsert(ctx, domain.Category{
		ProjectID: projectID,
		Key:       "child",
		Name:      "Child",
		ParentKey: root.Key,
	})
	if err != nil {
		t.Fatalf("upsert child: %v", err)
	}

	gin.SetMode(gin.TestMode)
	router, err := buildRouter(logDiscard(), pool, Deps{
		ProjectRepo: &stubProjectRepo{project: &domain.Project{ID: projectID, Key: "proj-key"}},
		ProductSvc:  &stubProductService{},
		CartSvc:     &stubCartService{},
		CategorySvc: catSvc,
		CustomerSvc: &stubCustomerService{customer: &domain.Customer{ID: "cust", ProjectID: projectID}},
	})
	if err != nil {
		t.Fatalf("build router: %v", err)
	}

	req := httptest.NewRequest("GET", "/proj-key/categories?limit=10&offset=0", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var resp ctCategoryList
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Total != 2 || resp.Count != 2 {
		t.Fatalf("unexpected totals %+v", resp)
	}
	var childResp *ctCategory
	for i := range resp.Results {
		if resp.Results[i].ID == child.ID {
			childResp = &resp.Results[i]
			break
		}
	}
	if childResp == nil {
		t.Fatalf("child category missing in response")
	}
	if childResp.Parent == nil || childResp.Parent.ID != root.ID {
		t.Fatalf("expected parent id %s, got %+v", root.ID, childResp.Parent)
	}
	if len(childResp.Ancestors) != 1 || childResp.Ancestors[0].ID != root.ID {
		t.Fatalf("expected ancestors to include root, got %+v", childResp.Ancestors)
	}
}

func categoriesPool(ctx context.Context, t *testing.T) *pgxpool.Pool {
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

func resetCategoryTables(ctx context.Context, t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	if _, err := pool.Exec(ctx, `TRUNCATE cart_lines, carts, products, customers, projects, categories RESTART IDENTITY CASCADE`); err != nil {
		t.Fatalf("truncate tables: %v", err)
	}
}
