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
