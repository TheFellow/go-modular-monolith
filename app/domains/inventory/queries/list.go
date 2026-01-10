package queries

import (
	"context"

	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/internal/dao"
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
)

func (q *Queries) List(ctx context.Context, filter dao.ListFilter) ([]*models.Stock, error) {
	return q.dao.List(ctx, filter)
}
