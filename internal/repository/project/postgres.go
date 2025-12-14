package project

import (
	"context"
	"errors"
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
			r.logger.Printf("project repo: key=%s not found", key)
			return nil, domain.ErrNotFound
		}
		r.logger.Printf("project repo: get key=%s error=%v", key, err)
		return nil, err
	}
	r.logger.Printf("project repo: key=%s id=%s name=%s", p.Key, p.ID, p.Name)
	return &p, nil
}

func (r *postgresRepo) Create(ctx context.Context, project *domain.Project) (*domain.Project, error) {
	const q = `
INSERT INTO projects (key, name)
VALUES ($1, $2)
RETURNING id::text, created_at
`
	var out domain.Project
	err := r.pool.QueryRow(ctx, q, project.Key, project.Name).Scan(&out.ID, &out.CreatedAt)
	if err != nil {
		r.logger.Printf("project repo: create key=%s error=%v", project.Key, err)
		return nil, err
	}
	out.Key = project.Key
	out.Name = project.Name
	r.logger.Printf("project repo: created key=%s id=%s", out.Key, out.ID)
	return &out, nil
}
