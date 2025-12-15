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
SELECT id::text, project_id::text, key, name, created_at
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
		if err := rows.Scan(&c.ID, &c.ProjectID, &c.Key, &c.Name, &c.CreatedAt); err != nil {
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
INSERT INTO categories (project_id, key, name)
VALUES ($1, $2, $3)
ON CONFLICT (project_id, key) DO UPDATE
SET name = EXCLUDED.name
RETURNING id::text, created_at
`
	var out domain.Category
	err := r.pool.QueryRow(ctx, q, c.ProjectID, c.Key, c.Name).Scan(&out.ID, &out.CreatedAt)
	if err != nil {
		return nil, err
	}
	out.ProjectID = c.ProjectID
	out.Key = c.Key
	out.Name = c.Name
	return &out, nil
}
