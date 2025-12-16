package category

import (
	"context"

	"commercetools-replica/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

type postgresRepo struct {
	pool *pgxpool.Pool
}

func NewPostgres(pool *pgxpool.Pool) Repository {
	return &postgresRepo{pool: pool}
}

func (r *postgresRepo) ListByProject(ctx context.Context, projectID string) ([]domain.Category, error) {
	const q = `
SELECT id::text, project_id::text, key, name, COALESCE(slug, ''), COALESCE(order_hint, ''), COALESCE(parent_key, ''), COALESCE(description, ''), COALESCE(meta_title, ''), COALESCE(meta_description, ''), created_at
FROM categories
WHERE project_id = $1
ORDER BY name ASC
`
	rows, err := r.pool.Query(ctx, q, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.Category
	for rows.Next() {
		var c domain.Category
		if err := rows.Scan(&c.ID, &c.ProjectID, &c.Key, &c.Name, &c.Slug, &c.OrderHint, &c.ParentKey, &c.Description, &c.MetaTitle, &c.MetaDescription, &c.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *postgresRepo) Upsert(ctx context.Context, c domain.Category) (*domain.Category, error) {
	const q = `
INSERT INTO categories (project_id, key, name, slug, order_hint, parent_key, description, meta_title, meta_description)
VALUES ($1, $2, $3, $4, NULLIF($5, ''), NULLIF($6, ''), $7, $8, $9)
ON CONFLICT (project_id, key) DO UPDATE
SET name = EXCLUDED.name,
    slug = COALESCE(NULLIF(EXCLUDED.slug, ''), categories.slug),
    order_hint = COALESCE(NULLIF(EXCLUDED.order_hint, ''), categories.order_hint),
    parent_key = COALESCE(NULLIF(EXCLUDED.parent_key, ''), categories.parent_key),
    description = COALESCE(NULLIF(EXCLUDED.description, ''), categories.description),
    meta_title = COALESCE(NULLIF(EXCLUDED.meta_title, ''), categories.meta_title),
    meta_description = COALESCE(NULLIF(EXCLUDED.meta_description, ''), categories.meta_description)
RETURNING id::text, created_at, COALESCE(slug, ''), COALESCE(order_hint, ''), COALESCE(parent_key, ''), COALESCE(description, ''), COALESCE(meta_title, ''), COALESCE(meta_description, '')
`
	var out domain.Category
	err := r.pool.QueryRow(ctx, q, c.ProjectID, c.Key, c.Name, c.Slug, c.OrderHint, c.ParentKey, c.Description, c.MetaTitle, c.MetaDescription).
		Scan(&out.ID, &out.CreatedAt, &out.Slug, &out.OrderHint, &out.ParentKey, &out.Description, &out.MetaTitle, &out.MetaDescription)
	if err != nil {
		return nil, err
	}
	out.ProjectID = c.ProjectID
	out.Key = c.Key
	out.Name = c.Name
	return &out, nil
}
