package product

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"

	"commercetools-replica/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type postgresRepo struct {
	pool   *pgxpool.Pool
	logger *log.Logger
}

func NewPostgres(pool *pgxpool.Pool, logger *log.Logger) Repository {
	if logger == nil {
		logger = log.New(io.Discard, "", 0)
	}
	return &postgresRepo{pool: pool, logger: logger}
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
		r.logger.Printf("product repo: list project_id=%s error=%v", projectID, err)
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
		r.logger.Printf("product repo: list rows project_id=%s error=%v", projectID, err)
		return nil, err
	}
	r.logger.Printf("product repo: list project_id=%s count=%d", projectID, len(result))
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
			r.logger.Printf("product repo: get project_id=%s id=%s not found", projectID, id)
			return nil, domain.ErrNotFound
		}
		r.logger.Printf("product repo: get project_id=%s id=%s error=%v", projectID, id, err)
		return nil, err
	}
	r.logger.Printf("product repo: get project_id=%s id=%s key=%s", projectID, id, p.Key)
	return &p, nil
}

func (r *postgresRepo) Upsert(ctx context.Context, product domain.Product) (*domain.Product, error) {
	const q = `
INSERT INTO products (id, project_id, key, sku, name, description, price_cents, currency, attributes)
VALUES (COALESCE(NULLIF($1, '')::uuid, gen_random_uuid()), $2, $3, $4, NULLIF($5, ''), $6, $7, $8, COALESCE($9, '{}'::jsonb))
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
		product.ID,
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
		r.logger.Printf("product repo: upsert key=%s project_id=%s error=%v", product.Key, product.ProjectID, err)
		return nil, err
	}
	if product.ID != "" && res.ID != product.ID {
		return nil, fmt.Errorf("product repo: id mismatch for key=%s project_id=%s existing_id=%s import_id=%s", product.Key, product.ProjectID, res.ID, product.ID)
	}
	res.ProjectID = product.ProjectID
	res.Key = product.Key
	res.SKU = product.SKU
	res.Name = product.Name
	res.Description = product.Description
	res.PriceCents = product.PriceCents
	res.Currency = product.Currency
	res.Attributes = product.Attributes
	r.logger.Printf("product repo: upserted key=%s project_id=%s id=%s", res.Key, res.ProjectID, res.ID)
	return &res, nil
}
