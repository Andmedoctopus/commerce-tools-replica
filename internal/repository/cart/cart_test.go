package cart

import (
	"context"
	"os"
	"testing"

	"commercetools-replica/internal/migrate"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgres_CreateAndGet(t *testing.T) {
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

	repo := NewPostgres(pool)
	created, err := repo.Create(ctx, CreateCartInput{
		ProjectID: projectID,
		Currency:  "USD",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.ProjectID != projectID || created.Currency != "USD" {
		t.Fatalf("unexpected cart %+v", created)
	}

	fetched, err := repo.GetByID(ctx, projectID, created.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if fetched.ID != created.ID || fetched.ProjectID != projectID {
		t.Fatalf("fetched mismatch %+v", fetched)
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
