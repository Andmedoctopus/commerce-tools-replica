package cart

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

func (r *postgresRepo) Create(ctx context.Context, in CreateCartInput) (*domain.Cart, error) {
	const q = `
INSERT INTO carts (project_id, customer_id, currency, total_cents, state)
VALUES ($1, $2, $3, 0, 'active')
RETURNING id::text, project_id::text, customer_id::text, currency, total_cents, state, created_at
`
	var cart domain.Cart
	var customerID *string
	if in.CustomerID != nil {
		customerID = in.CustomerID
	}
	if err := r.pool.QueryRow(ctx, q, in.ProjectID, customerID, in.Currency).Scan(
		&cart.ID,
		&cart.ProjectID,
		&customerID,
		&cart.Currency,
		&cart.TotalCents,
		&cart.State,
		&cart.CreatedAt,
	); err != nil {
		return nil, err
	}
	cart.CustomerID = customerID
	return &cart, nil
}

func (r *postgresRepo) GetByID(ctx context.Context, projectID, id string) (*domain.Cart, error) {
	const cartQuery = `
SELECT id::text, project_id::text, customer_id::text, currency, total_cents, state, created_at
FROM carts
WHERE project_id = $1 AND id = $2
`
	var cart domain.Cart
	var customerID *string
	err := r.pool.QueryRow(ctx, cartQuery, projectID, id).Scan(
		&cart.ID,
		&cart.ProjectID,
		&customerID,
		&cart.Currency,
		&cart.TotalCents,
		&cart.State,
		&cart.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	cart.CustomerID = customerID

	const linesQuery = `
SELECT id::text, cart_id::text, product_id::text, quantity, unit_price_cents, total_cents, snapshot, created_at
FROM cart_lines
WHERE cart_id = $1
ORDER BY created_at ASC
`
	rows, err := r.pool.Query(ctx, linesQuery, cart.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var line domain.CartLine
		if err := rows.Scan(
			&line.ID,
			&line.CartID,
			&line.ProductID,
			&line.Quantity,
			&line.UnitPriceCents,
			&line.TotalCents,
			&line.Snapshot,
			&line.CreatedAt,
		); err != nil {
			return nil, err
		}
		cart.Lines = append(cart.Lines, line)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &cart, nil
}
