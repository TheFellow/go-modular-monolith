package queries

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
)

func (q *Queries) Get(ctx store.Context, ingredientID entity.IngredientID) (*models.Inventory, error) {
	return q.dao.Get(ctx, ingredientID)
}
