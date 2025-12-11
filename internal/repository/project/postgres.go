package project

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

func (r *postgresRepo) GetByKey(ctx context.Context, key string) (*domain.Project, error) {
	const q = `
SELECT id::text, key, name, created_at
FROM projects
WHERE key = $1
`
	var p domain.Project
	err := r.pool.QueryRow(ctx, q, key).Scan(&p.ID, &p.Key, &p.Name, &p.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &p, nil
}
