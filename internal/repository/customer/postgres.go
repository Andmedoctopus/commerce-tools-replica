package customer

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"strings"

	"commercetools-replica/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type postgresRepo struct {
	pool   *pgxpool.Pool
	logger *log.Logger
}

// NewPostgres returns a Repository backed by Postgres.
func NewPostgres(pool *pgxpool.Pool, logger *log.Logger) Repository {
	if logger == nil {
		logger = log.New(io.Discard, "", 0)
	}
	return &postgresRepo{pool: pool, logger: logger}
}

func (r *postgresRepo) Create(ctx context.Context, c domain.Customer) (*domain.Customer, error) {
	addrJSON, err := json.Marshal(c.Addresses)
	if err != nil {
		return nil, err
	}
	shipJSON, err := json.Marshal(c.ShippingAddressIDs)
	if err != nil {
		return nil, err
	}
	billJSON, err := json.Marshal(c.BillingAddressIDs)
	if err != nil {
		return nil, err
	}

	const q = `
INSERT INTO customers (
    project_id, email, password_hash, first_name, last_name, date_of_birth, addresses,
    default_shipping_address_id, default_billing_address_id, shipping_address_ids, billing_address_ids
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
RETURNING id::text, project_id::text, email, password_hash, first_name, last_name, date_of_birth, addresses,
          default_shipping_address_id, default_billing_address_id, shipping_address_ids, billing_address_ids, created_at
`
	return r.scanCustomer(r.pool.QueryRow(
		ctx,
		q,
		c.ProjectID,
		strings.ToLower(c.Email),
		c.PasswordHash,
		c.FirstName,
		c.LastName,
		c.DateOfBirth,
		addrJSON,
		c.DefaultShippingAddressID,
		c.DefaultBillingAddressID,
		shipJSON,
		billJSON,
	))
}

func (r *postgresRepo) GetByEmail(ctx context.Context, projectID, email string) (*domain.Customer, error) {
	const q = `
SELECT id::text, project_id::text, email, password_hash, first_name, last_name, date_of_birth, addresses,
       default_shipping_address_id, default_billing_address_id, shipping_address_ids, billing_address_ids, created_at
FROM customers
WHERE project_id = $1 AND lower(email) = lower($2)
LIMIT 1
`
	return r.scanCustomer(r.pool.QueryRow(ctx, q, projectID, email))
}

func (r *postgresRepo) GetByID(ctx context.Context, projectID, id string) (*domain.Customer, error) {
	const q = `
SELECT id::text, project_id::text, email, password_hash, first_name, last_name, date_of_birth, addresses,
       default_shipping_address_id, default_billing_address_id, shipping_address_ids, billing_address_ids, created_at
FROM customers
WHERE project_id = $1 AND id = $2
LIMIT 1
`
	return r.scanCustomer(r.pool.QueryRow(ctx, q, projectID, id))
}

func (r *postgresRepo) scanCustomer(row pgx.Row) (*domain.Customer, error) {
	var c domain.Customer
	var addrJSON, shipJSON, billJSON []byte
	err := row.Scan(
		&c.ID,
		&c.ProjectID,
		&c.Email,
		&c.PasswordHash,
		&c.FirstName,
		&c.LastName,
		&c.DateOfBirth,
		&addrJSON,
		&c.DefaultShippingAddressID,
		&c.DefaultBillingAddressID,
		&shipJSON,
		&billJSON,
		&c.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, domain.ErrAlreadyExists
		}
		r.logger.Printf("customer repo: scan error=%v", err)
		return nil, err
	}
	if len(addrJSON) > 0 {
		if err := json.Unmarshal(addrJSON, &c.Addresses); err != nil {
			r.logger.Printf("customer repo: decode addresses id=%s err=%v", c.ID, err)
			return nil, err
		}
	}
	if len(shipJSON) > 0 {
		if err := json.Unmarshal(shipJSON, &c.ShippingAddressIDs); err != nil {
			r.logger.Printf("customer repo: decode shipping ids id=%s err=%v", c.ID, err)
			return nil, err
		}
	}
	if len(billJSON) > 0 {
		if err := json.Unmarshal(billJSON, &c.BillingAddressIDs); err != nil {
			r.logger.Printf("customer repo: decode billing ids id=%s err=%v", c.ID, err)
			return nil, err
		}
	}
	return &c, nil
}
