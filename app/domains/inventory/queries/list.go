package queries

import (
	"context"

	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
)

func (q *Queries) List(ctx context.Context) ([]models.Stock, error) {
	out, err := q.dao.List(ctx)
	if err != nil {
		return nil, errors.Internalf("list stock: %w", err)
	}
	return out, nil
}
