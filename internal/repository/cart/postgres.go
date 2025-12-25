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
INSERT INTO carts (project_id, customer_id, anonymous_id, currency, total_cents, state)
VALUES ($1, $2, $3, $4, 0, 'active')
RETURNING id::text, project_id::text, customer_id::text, anonymous_id::text, currency, total_cents, state, created_at
`
	var cart domain.Cart
	var customerID *string
	var anonymousID *string
	if in.CustomerID != nil {
		customerID = in.CustomerID
	}
	if in.AnonymousID != nil {
		anonymousID = in.AnonymousID
	}
	if err := r.pool.QueryRow(ctx, q, in.ProjectID, customerID, anonymousID, in.Currency).Scan(
		&cart.ID,
		&cart.ProjectID,
		&customerID,
		&anonymousID,
		&cart.Currency,
		&cart.TotalCents,
		&cart.State,
		&cart.CreatedAt,
	); err != nil {
		return nil, err
	}
	cart.CustomerID = customerID
	cart.AnonymousID = anonymousID
	return &cart, nil
}

func (r *postgresRepo) GetByID(ctx context.Context, projectID, id string) (*domain.Cart, error) {
	const cartQuery = `
SELECT id::text, project_id::text, customer_id::text, anonymous_id::text, currency, total_cents, state, created_at
FROM carts
WHERE project_id = $1 AND id = $2
`
	return r.fetchCart(ctx, cartQuery, projectID, id)
}

func (r *postgresRepo) GetActiveByCustomer(ctx context.Context, projectID, customerID string) (*domain.Cart, error) {
	const cartQuery = `
SELECT id::text, project_id::text, customer_id::text, anonymous_id::text, currency, total_cents, state, created_at
FROM carts
WHERE project_id = $1 AND customer_id = $2 AND state = 'active'
ORDER BY created_at DESC
LIMIT 1
`
	return r.fetchCart(ctx, cartQuery, projectID, customerID)
}

func (r *postgresRepo) GetActiveByAnonymous(ctx context.Context, projectID, anonymousID string) (*domain.Cart, error) {
	const cartQuery = `
SELECT id::text, project_id::text, customer_id::text, anonymous_id::text, currency, total_cents, state, created_at
FROM carts
WHERE project_id = $1 AND anonymous_id = $2 AND state = 'active'
ORDER BY created_at DESC
LIMIT 1
`
	return r.fetchCart(ctx, cartQuery, projectID, anonymousID)
}

func (r *postgresRepo) AssignCustomerToAnonymous(ctx context.Context, projectID, anonymousID, customerID string) (*domain.Cart, error) {
	const q = `
UPDATE carts
SET customer_id = $1,
    anonymous_id = NULL
WHERE project_id = $2 AND anonymous_id = $3 AND state = 'active'
RETURNING id::text
`
	var cartID string
	if err := r.pool.QueryRow(ctx, q, customerID, projectID, anonymousID).Scan(&cartID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return r.fetchCart(ctx, `
SELECT id::text, project_id::text, customer_id::text, anonymous_id::text, currency, total_cents, state, created_at
FROM carts
WHERE id = $1
`, cartID)
}

func (r *postgresRepo) AddLineItem(ctx context.Context, cartID string, product domain.Product, quantity int, snapshot map[string]interface{}) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var lineID string
	var existingQty int
	var unitPrice int64
	err = tx.QueryRow(ctx, `
SELECT id::text, quantity, unit_price_cents
FROM cart_lines
WHERE cart_id = $1 AND product_id = $2
`, cartID, product.ID).Scan(&lineID, &existingQty, &unitPrice)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return err
	}

	if err == nil {
		newQty := existingQty + quantity
		newTotal := unitPrice * int64(newQty)
		if _, err := tx.Exec(ctx, `
UPDATE cart_lines
SET quantity = $1, total_cents = $2
WHERE id = $3
`, newQty, newTotal, lineID); err != nil {
			return err
		}
	} else {
		unitPrice = product.PriceCents
		total := unitPrice * int64(quantity)
		if _, err := tx.Exec(ctx, `
INSERT INTO cart_lines (cart_id, product_id, quantity, unit_price_cents, total_cents, snapshot)
VALUES ($1, $2, $3, $4, $5, $6)
`, cartID, product.ID, quantity, unitPrice, total, snapshot); err != nil {
			return err
		}
	}

	if err := updateCartTotal(ctx, tx, cartID); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *postgresRepo) ChangeLineItemQuantity(ctx context.Context, cartID, lineItemID string, quantity int) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if quantity <= 0 {
		cmd, err := tx.Exec(ctx, `
DELETE FROM cart_lines
WHERE id = $1 AND cart_id = $2
`, lineItemID, cartID)
		if err != nil {
			return err
		}
		if cmd.RowsAffected() == 0 {
			return domain.ErrNotFound
		}
	} else {
		var unitPrice int64
		err := tx.QueryRow(ctx, `
SELECT unit_price_cents
FROM cart_lines
WHERE id = $1 AND cart_id = $2
`, lineItemID, cartID).Scan(&unitPrice)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return domain.ErrNotFound
			}
			return err
		}
		total := unitPrice * int64(quantity)
		if _, err := tx.Exec(ctx, `
UPDATE cart_lines
SET quantity = $1, total_cents = $2
WHERE id = $3 AND cart_id = $4
`, quantity, total, lineItemID, cartID); err != nil {
			return err
		}
	}

	if err := updateCartTotal(ctx, tx, cartID); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *postgresRepo) fetchCart(ctx context.Context, cartQuery string, args ...interface{}) (*domain.Cart, error) {
	var cart domain.Cart
	var customerID *string
	var anonymousID *string
	err := r.pool.QueryRow(ctx, cartQuery, args...).Scan(
		&cart.ID,
		&cart.ProjectID,
		&customerID,
		&anonymousID,
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
	cart.AnonymousID = anonymousID

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

func updateCartTotal(ctx context.Context, tx pgx.Tx, cartID string) error {
	_, err := tx.Exec(ctx, `
UPDATE carts
SET total_cents = COALESCE((
	SELECT SUM(total_cents)
	FROM cart_lines
	WHERE cart_id = $1
), 0)
WHERE id = $1
`, cartID)
	return err
}
