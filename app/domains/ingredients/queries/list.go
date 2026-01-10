package queries

import (
	"context"

	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/internal/dao"
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
)

func (q *Queries) List(ctx context.Context, filter dao.ListFilter) ([]*models.Ingredient, error) {
	return q.dao.List(ctx, filter)
}
