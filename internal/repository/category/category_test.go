package category

import (
	"context"
	"testing"

	"commercetools-replica/internal/domain"
	"commercetools-replica/internal/migrate"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgres_UpsertAndList(t *testing.T) {
	ctx := context.Background()
	pool := testPool(ctx, t)
	defer pool.Close()

	if err := migrate.Apply(ctx, pool); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	resetTables(ctx, t, pool)

	var projectID string
	if err := pool.QueryRow(ctx, `INSERT INTO projects (key, name) VALUES ('proj-key', 'Proj') RETURNING id::text`).Scan(&projectID); err != nil {
		t.Fatalf("insert project: %v", err)
	}

	repo := NewPostgres(pool)
	cat, err := repo.Upsert(ctx, domain.Category{
		ProjectID: projectID,
		Key:       "cat-1",
		Name:      "Cat 1",
		Slug:      "cat-1",
		ParentKey: "",
	})
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}
	if cat.ID == "" || cat.Key != "cat-1" {
		t.Fatalf("unexpected category %+v", cat)
	}

	list, err := repo.ListByProject(ctx, projectID)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 1 || list[0].Key != "cat-1" {
		t.Fatalf("unexpected list %+v", list)
	}
}

func TestPostgres_UpsertUpdatesExisting(t *testing.T) {
	ctx := context.Background()
	pool := testPool(ctx, t)
	defer pool.Close()

	if err := migrate.Apply(ctx, pool); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	resetTables(ctx, t, pool)

	var projectID string
	if err := pool.QueryRow(ctx, `INSERT INTO projects (key, name) VALUES ('proj-key', 'Proj') RETURNING id::text`).Scan(&projectID); err != nil {
		t.Fatalf("insert project: %v", err)
	}

	repo := NewPostgres(pool)
	first, err := repo.Upsert(ctx, domain.Category{
		ProjectID: projectID,
		Key:       "cat-1",
		Name:      "Cat 1",
		Slug:      "cat-1",
	})
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}

	second, err := repo.Upsert(ctx, domain.Category{
		ProjectID: projectID,
		Key:       "cat-1",
		Name:      "Cat 1 Updated",
		Slug:      "cat-1",
	})
	if err != nil {
		t.Fatalf("upsert update: %v", err)
	}
	if second.ID != first.ID {
		t.Fatalf("expected same ID after update")
	}
	if second.Name != "Cat 1 Updated" {
		t.Fatalf("expected updated name, got %+v", second)
	}
}

func testPool(ctx context.Context, t *testing.T) *pgxpool.Pool {
	t.Helper()
	candidates := []string{
		"postgres://commerce:commerce@db-test:5432/commerce_test?sslmode=disable",
		"postgres://commerce:commerce@localhost:5433/commerce_test?sslmode=disable",
	}
	var lastErr error
	for _, dsn := range candidates {
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

func resetTables(ctx context.Context, t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	if _, err := pool.Exec(ctx, `TRUNCATE cart_lines, carts, products, customers, projects, categories RESTART IDENTITY CASCADE`); err != nil {
		t.Fatalf("truncate tables: %v", err)
	}
}
