package queries

import (
	ingredientsdao "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/internal/dao"
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
)

func (q *Queries) List(ctx store.Context, filter ingredientsdao.ListFilter) ([]*models.Ingredient, error) {
	return q.dao.List(ctx, filter)
}
