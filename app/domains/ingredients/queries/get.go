package queries

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
)

func (q *Queries) Get(ctx store.Context, id entity.IngredientID) (*models.Ingredient, error) {
	return q.dao.Get(ctx, id)
}
