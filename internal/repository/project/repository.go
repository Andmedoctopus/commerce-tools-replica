package project

import (
	"context"

	"commercetools-replica/internal/domain"
)

type Repository interface {
	GetByKey(ctx context.Context, key string) (*domain.Project, error)
}
