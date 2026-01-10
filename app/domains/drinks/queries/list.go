package queries

import (
	"context"

	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/internal/dao"
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
)

func (q *Queries) List(ctx context.Context, filter dao.ListFilter) ([]*models.Drink, error) {
	return q.dao.List(ctx, filter)
}
