package token

import (
	"context"
	"errors"

	"commercetools-replica/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type postgresRepo struct {
	pool *pgxpool.Pool
}

func NewPostgres(pool *pgxpool.Pool) Repository {
	return &postgresRepo{pool: pool}
}

func (r *postgresRepo) Create(ctx context.Context, token Token) error {
	const q = `
INSERT INTO tokens (token, project_id, customer_id, anonymous_id, kind, expires_at)
VALUES ($1, $2, $3, $4, $5, $6)
`
	_, err := r.pool.Exec(ctx, q, token.Token, token.ProjectID, token.CustomerID, token.AnonymousID, token.Kind, token.ExpiresAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return domain.ErrAlreadyExists
		}
		return err
	}
	return nil
}

func (r *postgresRepo) Get(ctx context.Context, token string) (*Token, error) {
	const q = `
SELECT token, project_id::text, customer_id::text, anonymous_id, kind, expires_at, created_at
FROM tokens
WHERE token = $1
LIMIT 1
`
	var out Token
	var customerID *string
	var anonymousID *string
	if err := r.pool.QueryRow(ctx, q, token).Scan(
		&out.Token,
		&out.ProjectID,
		&customerID,
		&anonymousID,
		&out.Kind,
		&out.ExpiresAt,
		&out.CreatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	out.CustomerID = customerID
	out.AnonymousID = anonymousID
	return &out, nil
}

func (r *postgresRepo) Delete(ctx context.Context, token string) error {
	cmd, err := r.pool.Exec(ctx, `DELETE FROM tokens WHERE token = $1`, token)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}
