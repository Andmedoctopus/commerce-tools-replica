package product

import (
	"context"
	"os"
	"testing"

	"commercetools-replica/internal/migrate"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgres_ListAndGet(t *testing.T) {
	ctx := context.Background()
	pool := testPool(ctx, t)
	defer pool.Close()

	if err := migrate.Apply(ctx, pool); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	resetTables(ctx, t, pool)

	var projectID string
	err := pool.QueryRow(ctx, `INSERT INTO projects (key, name) VALUES (gen_random_uuid()::text, 'Proj') RETURNING id::text`).Scan(&projectID)
	if err != nil {
		t.Fatalf("insert project: %v", err)
	}

	var pid string
	err = pool.QueryRow(ctx, `
		INSERT INTO products (project_id, key, sku, name, description, price_cents, currency, attributes)
		VALUES ($1, 'p1', 'SKU1', 'Prod 1', 'desc', 100, 'USD', '{}'::jsonb)
		RETURNING id::text
	`, projectID).Scan(&pid)
	if err != nil {
		t.Fatalf("insert product: %v", err)
	}

	repo := NewPostgres(pool)

	list, err := repo.ListByProject(ctx, projectID)
	if err != nil {
		t.Fatalf("ListByProject: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 product, got %d", len(list))
	}

	got, err := repo.GetByID(ctx, projectID, pid)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.ID != pid || got.ProjectID != projectID {
		t.Fatalf("unexpected product %+v", got)
	}
}

func testPool(ctx context.Context, t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("TEST_DB_DSN")
	if dsn == "" {
		dsn = "postgres://commerce:commerce@db-test:5432/commerce_test?sslmode=disable"
	}
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect db: %v", err)
	}
	return pool
}

func resetTables(ctx context.Context, t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	if _, err := pool.Exec(ctx, `TRUNCATE cart_lines, carts, products, customers, projects RESTART IDENTITY CASCADE`); err != nil {
		t.Fatalf("truncate tables: %v", err)
	}
}
