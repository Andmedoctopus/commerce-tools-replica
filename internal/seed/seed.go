package seed

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type productSeed struct {
	Key         string
	SKU         string
	Name        string
	Description string
	PriceCents  int64
	Currency    string
}

// Apply inserts basic seed data for manual testing. It is idempotent via ON CONFLICT.
func Apply(ctx context.Context, pool *pgxpool.Pool) error {
	projectID, err := ensureProject(ctx, pool, "demo", "Demo Project")
	if err != nil {
		return fmt.Errorf("ensure project: %w", err)
	}

	products := []productSeed{
		{
			Key:         "demo-shirt",
			SKU:         "SKU-DEMO-TSHIRT",
			Name:        "Demo T-Shirt",
			Description: "Soft cotton tee for demo purposes",
			PriceCents:  1999,
			Currency:    "USD",
		},
		{
			Key:         "demo-mug",
			SKU:         "SKU-DEMO-MUG",
			Name:        "Demo Mug",
			Description: "Ceramic mug with demo logo",
			PriceCents:  1299,
			Currency:    "USD",
		},
	}

	for _, p := range products {
		if err := upsertProduct(ctx, pool, projectID, p); err != nil {
			return fmt.Errorf("upsert product %s: %w", p.Key, err)
		}
	}

	return nil
}

func ensureProject(ctx context.Context, pool *pgxpool.Pool, key, name string) (string, error) {
	const q = `
INSERT INTO projects (key, name)
VALUES ($1, $2)
ON CONFLICT (key) DO UPDATE SET name = EXCLUDED.name
RETURNING id::text
`
	var id string
	if err := pool.QueryRow(ctx, q, key, name).Scan(&id); err != nil {
		return "", err
	}
	return id, nil
}

func upsertProduct(ctx context.Context, pool *pgxpool.Pool, projectID string, p productSeed) error {
	const q = `
INSERT INTO products (project_id, key, sku, name, description, price_cents, currency, attributes)
VALUES ($1, $2, $3, $4, $5, $6, $7, '{}'::jsonb)
ON CONFLICT (project_id, key) DO UPDATE
SET sku = EXCLUDED.sku,
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    price_cents = EXCLUDED.price_cents,
    currency = EXCLUDED.currency
`
	_, err := pool.Exec(ctx, q, projectID, p.Key, p.SKU, p.Name, p.Description, p.PriceCents, p.Currency)
	if err != nil {
		return err
	}
	return nil
}
