package product

import (
	"context"
	"errors"

	"commercetools-replica/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type postgresRepo struct {
	pool *pgxpool.Pool
}

func NewPostgres(pool *pgxpool.Pool) Repository {
	return &postgresRepo{pool: pool}
}

func (r *postgresRepo) ListByProject(ctx context.Context, projectID string) ([]domain.Product, error) {
	const q = `
SELECT id::text, project_id::text, key, sku, name, COALESCE(description, ''), price_cents, currency, attributes, created_at
FROM products
WHERE project_id = $1
ORDER BY created_at DESC
`
	rows, err := r.pool.Query(ctx, q, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.Product
	for rows.Next() {
		var p domain.Product
		if err := rows.Scan(&p.ID, &p.ProjectID, &p.Key, &p.SKU, &p.Name, &p.Description, &p.PriceCents, &p.Currency, &p.Attributes, &p.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *postgresRepo) GetByID(ctx context.Context, projectID, id string) (*domain.Product, error) {
	const q = `
SELECT id::text, project_id::text, key, sku, name, COALESCE(description, ''), price_cents, currency, attributes, created_at
FROM products
WHERE project_id = $1 AND id = $2
`
	var p domain.Product
	err := r.pool.QueryRow(ctx, q, projectID, id).Scan(&p.ID, &p.ProjectID, &p.Key, &p.SKU, &p.Name, &p.Description, &p.PriceCents, &p.Currency, &p.Attributes, &p.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &p, nil
}

func (r *postgresRepo) Upsert(ctx context.Context, product domain.Product) (*domain.Product, error) {
	const q = `
INSERT INTO products (project_id, key, sku, name, description, price_cents, currency, attributes)
VALUES ($1, $2, $3, $4, NULLIF($5, ''), $6, $7, COALESCE($8, '{}'::jsonb))
ON CONFLICT (project_id, key) DO UPDATE SET
    sku = EXCLUDED.sku,
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    price_cents = EXCLUDED.price_cents,
    currency = EXCLUDED.currency,
    attributes = EXCLUDED.attributes
RETURNING id::text, created_at
`
	var res domain.Product
	err := r.pool.QueryRow(ctx, q,
		product.ProjectID,
		product.Key,
		product.SKU,
		product.Name,
		product.Description,
		product.PriceCents,
		product.Currency,
		product.Attributes,
	).Scan(&res.ID, &res.CreatedAt)
	if err != nil {
		return nil, err
	}
	res.ProjectID = product.ProjectID
	res.Key = product.Key
	res.SKU = product.SKU
	res.Name = product.Name
	res.Description = product.Description
	res.PriceCents = product.PriceCents
	res.Currency = product.Currency
	res.Attributes = product.Attributes
	return &res, nil
}
