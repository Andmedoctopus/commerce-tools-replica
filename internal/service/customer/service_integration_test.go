package customer

import (
	"context"
	"log"
	"os"
	"testing"

	"commercetools-replica/internal/migrate"
	customerrepo "commercetools-replica/internal/repository/customer"
	tokenrepo "commercetools-replica/internal/repository/token"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestSignupAndLogin_Integration(t *testing.T) {
	ctx := context.Background()
	pool := integrationPool(ctx, t)
	defer pool.Close()

	if err := migrate.Apply(ctx, pool); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	resetTables(ctx, t, pool)

	var projectID string
	if err := pool.QueryRow(ctx, `INSERT INTO projects (key, name) VALUES ('proj-key', 'Proj') RETURNING id::text`).Scan(&projectID); err != nil {
		t.Fatalf("insert project: %v", err)
	}

	repo := customerrepo.NewPostgres(pool, log.New(os.Stdout, "[test] ", log.LstdFlags))
	tokenRepo := tokenrepo.NewPostgres(pool)
	svc := New(repo, tokenRepo)

	password := "Abcdefg1"
	cust, err := svc.Signup(ctx, projectID, SignupInput{
		Email:     "integration@example.com",
		Password:  password,
		FirstName: "Int",
		LastName:  "User",
		Addresses: []AddressInput{
			{Country: "US", StreetName: "Main", PostalCode: "00000", City: "Testville"},
		},
		DefaultShippingAddress: intPtr(0),
		DefaultBillingAddress:  intPtr(0),
	})
	if err != nil {
		t.Fatalf("signup: %v", err)
	}
	if cust == nil || cust.ID == "" {
		t.Fatalf("expected created customer, got %+v", cust)
	}

	_, access, refresh, err := svc.Login(ctx, projectID, "integration@example.com", password)
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if access == "" || refresh == "" {
		t.Fatalf("expected tokens, got access=%q refresh=%q", access, refresh)
	}
}

func integrationPool(ctx context.Context, t *testing.T) *pgxpool.Pool {
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

func resetTables(ctx context.Context, t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	if _, err := pool.Exec(ctx, `TRUNCATE cart_lines, carts, products, customers, projects RESTART IDENTITY CASCADE`); err != nil {
		t.Fatalf("truncate tables: %v", err)
	}
}

func intPtr(v int) *int {
	return &v
}
