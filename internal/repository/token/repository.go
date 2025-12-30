package token

import (
	"context"
	"time"
)

type Token struct {
	Token       string
	ProjectID   string
	CustomerID  *string
	AnonymousID *string
	Kind        string
	ExpiresAt   time.Time
	CreatedAt   time.Time
}

type Repository interface {
	Create(ctx context.Context, token Token) error
	Get(ctx context.Context, token string) (*Token, error)
	Delete(ctx context.Context, token string) error
}
